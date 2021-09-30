package manager

import (
	"context"
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"
)

type Zone struct {
	Owner             string
	Name              string
	ID                string
	HostId            string
	Password          string
	ConnectedMembers  []string
	Status            string
	AuthorizedMembers []string
}

type ZoneManager struct {
	DB          *ZoneDBManager
	manager     *Manager
	authManager *AuthManager
}

func NewZoneManager(manager *Manager, authManager *AuthManager) (zoneManager *ZoneManager, err error) {
	zoneDBManager, err := NewZoneDBManager("localhost", 27017)
	if err != nil {
		return
	}
	zoneManager = &ZoneManager{
		DB:          zoneDBManager,
		manager:     manager,
		authManager: authManager,
	}
	return
}

func (zm *ZoneManager) GetZoneByOwner(token string, owner string, lastIndex int64) (zones []*Zone, err error) {
	if _, ok := zm.authManager.AuthTokenValid[token]; !ok {
		err = fmt.Errorf("not a valid token provided")
		return
	}
	if zm.authManager.AuthTokenValid[token] != owner {
		err = fmt.Errorf("invalid access")
		return
	}
	zones, err = zm.DB.GetZonesByOwner(context.Background(), owner, 100, lastIndex)
	return
}

func (zm *ZoneManager) ConnectToZone(token string, zoneId string, from string, password string) (err error) {
	if _, ok := zm.authManager.AuthTokenValid[token]; !ok {
		err = fmt.Errorf("not a valid token provided")
		return
	}
	if zm.authManager.AuthTokenValid[token] != from {
		err = fmt.Errorf("invalid access")
		return
	}
	var zone *Zone
	if zone, err = zm.DB.GetZone(context.Background(), zoneId); err != nil {
		return
	}
	var contains bool = false
	for _, am := range zone.AuthorizedMembers {
		if am == from {
			contains = true
			break
		}
	}
	if contains {
		signalIncomingZoneMember := func() {
			for _, member := range zone.ConnectedMembers {
				if member != from {
					if _, ok := zm.manager.GRPCPeers[member]; ok {
						if err := zm.manager.GRPCPeers[member].Conn.Send(&Response{
							Type:    INCOMING_ZONE_MEMBER,
							Success: true,
							Payload: map[string]string{
								"id": from,
							},
						}); err != nil {
							delete(zm.manager.GRPCPeers, member)
							return
						}
					} else if _, ok := zm.manager.WSPeers[member]; ok {
						zm.manager.WSPeers[member].mux.Lock()
						if err = zm.manager.WSPeers[member].Conn.WriteJSON(map[string]interface{}{
							"from":    from,
							"to":      member,
							"type":    INCOMING_ZONE_MEMBER,
							"payload": map[string]string{},
						}); err != nil {
							log.Println(err)
							zm.manager.WSPeers[member].mux.Unlock()
							return
						}
						zm.manager.WSPeers[member].mux.Unlock()
					}
				}
			}
		}
		err = zm.DB.UpdateZoneMembers(context.Background(), zone.ID, append(zone.ConnectedMembers, from))
		if err == nil {
			go signalIncomingZoneMember()
		}
	}
	return
}

func (zm *ZoneManager) ListAllZones(lastIndex int64) (zones []*Zone, err error) {
	zones, err = zm.DB.GetZones(context.Background(), 100, lastIndex)
	return
}

func (zm *ZoneManager) ListZonesByName(lastIndex int64, zoneName string) (zones []*Zone, err error) {
	zones, err = zm.DB.GetZonesByName(context.Background(), zoneName, 100, lastIndex)
	return
}

func (zm *ZoneManager) ListZonesByHost(lastIndex int64, host string) (zones []*Zone, err error) {
	zones, err = zm.DB.GetZonesByHost(context.Background(), host, 100, lastIndex)
	return
}

func (zm *ZoneManager) ListZonesByID(lastIndex int64, zoneId string) (zones []*Zone, err error) {
	zones, err = zm.DB.GetZonesByID(context.Background(), zoneId, 100, lastIndex)
	return
}

func (zm *ZoneManager) GetZoneByID(zoneId string) (zones *Zone, err error) {
	zones, err = zm.DB.GetZoneByID(context.Background(), zoneId)
	return
}

func (zm *ZoneManager) LeaveZone(token string,zoneId string, from string) (err error) {
	if _, ok := zm.authManager.AuthTokenValid[token]; !ok {
		err = fmt.Errorf("not a valid token provided")
		return
	}
	if zm.authManager.AuthTokenValid[token] != from {
		err = fmt.Errorf("invalid access")
		return
	}
	var zone *Zone
	if zone, err = zm.DB.GetZone(context.Background(), zoneId); err != nil {
		return
	}
	var memberIndex int
	for i, member := range zone.ConnectedMembers {
		if member == from {
			memberIndex = i
			break
		}
	}
	if len(zone.ConnectedMembers) < 2 {
		zone.ConnectedMembers = make([]string, 0)
	} else {
		zone.ConnectedMembers = append(zone.ConnectedMembers[:memberIndex], zone.ConnectedMembers[memberIndex+1:]...)
	}
	if err = zm.DB.UpdateZoneMembers(context.Background(),zoneId,zone.ConnectedMembers); err != nil {
		return
	}
	signalLeaving := func(to []string) {
		for _, member := range to {
			if member != from {
				if _, ok := zm.manager.GRPCPeers[member]; ok {
					if wserr := zm.manager.GRPCPeers[member].Conn.Send(&Response{
						Type:    ZONE_MEMBER_DISCONNECTED,
						Success: true,
						Payload: map[string]string{
							"from":   from,
							"zoneId": zoneId,
						},
					}); wserr != nil {
						delete(zm.manager.GRPCPeers, member)
						continue
					}
				} else if _, ok := zm.manager.WSPeers[member]; ok {
					zm.manager.WSPeers[member].mux.Lock()
					if err = zm.manager.WSPeers[member].Conn.WriteJSON(map[string]interface{}{
						"from": from,
						"to":   member,
						"type": ZONE_MEMBER_DISCONNECTED,
						"payload": map[string]string{
							"zoneId": zoneId,
						},
					}); err != nil {
						log.Println(err)
						zm.manager.WSPeers[member].mux.Unlock()
						continue
					}
					zm.manager.WSPeers[member].mux.Unlock()
				}
			}
		}
	}
	go signalLeaving(zone.ConnectedMembers)
	go signalLeaving(zone.AuthorizedMembers)
	return
}

func (zm *ZoneManager) CreateZone(token string, zoneId string, owner string, zoneName string, password string, zoneHost string) (err error) {
	if _, ok := zm.authManager.AuthTokenValid[token]; !ok {
		err = fmt.Errorf("not a valid token provided")
		return
	}
	if zm.authManager.AuthTokenValid[token] != owner {
		err = fmt.Errorf("invalid access")
		return
	}
	var zonePass string
	if output, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost); err != nil {
		return err
	} else {
		zonePass = string(output)
	}
	zone := Zone{
		Owner:             owner,
		ID:                zoneId,
		Name:              zoneName,
		HostId:            zoneHost,
		Password:          zonePass,
		ConnectedMembers:  make([]string, 0),
		AuthorizedMembers: make([]string, 0),
		Status:            "online",
	}
	if err = zm.DB.AddNewZone(context.Background(), &zone); err != nil {
		return
	}
	if err = zm.UpdateZoneAuthorizedMembers(zoneId, owner); err != nil {
		return
	}
	if _, ok := zm.manager.GRPCPeers[zoneHost]; ok {
		_ = zm.manager.GRPCPeers[zoneHost].Conn.Send(&Response{
			Type:    NEW_ZONE,
			Success: true,
			Payload: map[string]string{
				"ID": zoneId,
			},
		})

	}
	return
}

func (zm *ZoneManager) DeleteZone(token string, zoneId string, from string) (err error) {
	if _, ok := zm.authManager.AuthTokenValid[token]; !ok {
		err = fmt.Errorf("not a valid token provided")
		return
	}
	if zm.authManager.AuthTokenValid[token] != from {
		err = fmt.Errorf("invalid access")
		return
	}
	zone, zoneErr := zm.DB.GetZone(context.Background(), zoneId)
	if zoneErr != nil {
		err = fmt.Errorf("this zone does not exist")
		return
	}
	for _, member := range zone.AuthorizedMembers {
		if err = zm.DeleteZoneAuthorizedMembers(zoneId, member); err != nil {
			return
		}
	}
	err = zm.DB.DeleteZone(context.Background(), zoneId)
	return
}

func (zm *ZoneManager) ModifyZone(token string, zoneId string, from string, zoneName string, password string) (err error) {
	return
}

func (zm *ZoneManager) UpdateZoneName(zoneId string, zoneName string) (err error) {
	err = zm.DB.UpdateZoneName(context.Background(), zoneId, zoneName)
	return
}

func (zm *ZoneManager) UpdateZonePassword(zoneId string, password string) error {
	pass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return zm.DB.UpdateZonePassword(context.Background(), zoneId, string(pass))
}

func (zm *ZoneManager) UpdateZoneAuthorizedMembers(zoneId string, authorizedMember string) (err error) {
	var zone *Zone
	if zone, err = zm.DB.GetZone(context.Background(), zoneId); err != nil {
		return
	}
	var peer *Peer
	if peer, err = zm.manager.PeerManager.GetPeer(authorizedMember); err != nil {
		return
	}
	for _, am := range zone.AuthorizedMembers {
		if am == authorizedMember {
			err = fmt.Errorf("user already authorized")
			return
		}
	}
	if err = zm.DB.UpdateZoneAuthorizedMembers(context.Background(), zoneId, append(zone.AuthorizedMembers, authorizedMember)); err != nil {
		return
	}
	signalNewAuthorizedMember := func() {
		if _, ok := zm.manager.GRPCPeers[authorizedMember]; ok {
			if err := zm.manager.GRPCPeers[authorizedMember].Conn.Send(&Response{
				Type:    NEW_AUTHORIZED_ZONE,
				Success: true,
				Payload: map[string]string{
					"zoneId": zoneId,
				},
			}); err != nil {
				delete(zm.manager.GRPCPeers, authorizedMember)
				return
			}
		} else if _, ok := zm.manager.WSPeers[authorizedMember]; ok {
			zm.manager.WSPeers[authorizedMember].mux.Lock()
			if err = zm.manager.WSPeers[authorizedMember].Conn.WriteJSON(map[string]interface{}{
				"from": "server",
				"to":   authorizedMember,
				"type": NEW_AUTHORIZED_ZONE,
				"payload": map[string]string{
					"zoneId": zoneId,
				},
			}); err != nil {
				log.Println(err)
				zm.manager.WSPeers[authorizedMember].mux.Unlock()
				return
			}
			zm.manager.WSPeers[authorizedMember].mux.Unlock()
		}
	}
	for _, v := range peer.KnownZonesId {
		if v == zoneId {
			err = fmt.Errorf("zone already known")
			return
		}
	}
	if err = zm.manager.PeerManager.DB.UpdateKnownZones(context.Background(), authorizedMember, append(peer.KnownZonesId, zoneId)); err != nil {
		return
	}
	go signalNewAuthorizedMember()
	return
}

func (zm *ZoneManager) DeleteZoneAuthorizedMembers(zoneId string, authorizedMember string) (err error) {
	var zone *Zone
	if zone, err = zm.DB.GetZone(context.Background(), zoneId); err != nil {
		return
	}
	var peer *Peer
	if peer, err = zm.manager.PeerManager.GetPeer(authorizedMember); err != nil {
		return
	}
	var index int
	var contain bool
	for i, am := range zone.AuthorizedMembers {
		if am == authorizedMember {
			index = i
			contain = true
			break
		}
	}
	if !contain {
		err = fmt.Errorf("member not in zone")
		return
	}
	if len(zone.AuthorizedMembers) < 2 {
		if err = zm.DB.UpdateZoneAuthorizedMembers(context.Background(), zoneId, make([]string, 0)); err != nil {
			return
		}
	} else {
		if err = zm.DB.UpdateZoneAuthorizedMembers(context.Background(), zoneId, append(zone.AuthorizedMembers[:index], zone.AuthorizedMembers[index+1:]...)); err != nil {
			return
		}
	}
	_, _ = index, contain
	signalRemovedAuthorizedMember := func() {
		if _, ok := zm.manager.GRPCPeers[authorizedMember]; ok {
			if err := zm.manager.GRPCPeers[authorizedMember].Conn.Send(&Response{
				Type:    REMOVED_AUTHORIZED_ZONE,
				Success: true,
				Payload: map[string]string{
					"zoneId": zoneId,
				},
			}); err != nil {
				delete(zm.manager.GRPCPeers, authorizedMember)
				return
			}
		} else if _, ok := zm.manager.WSPeers[authorizedMember]; ok {
			zm.manager.WSPeers[authorizedMember].mux.Lock()
			if err = zm.manager.WSPeers[authorizedMember].Conn.WriteJSON(map[string]interface{}{
				"from": "server",
				"to":   authorizedMember,
				"type": REMOVED_AUTHORIZED_ZONE,
				"payload": map[string]string{
					"zoneId": zoneId,
				},
			}); err != nil {
				log.Println(err)
				zm.manager.WSPeers[authorizedMember].mux.Unlock()
				return
			}
			zm.manager.WSPeers[authorizedMember].mux.Unlock()
		}
	}
	var kIndex int
	var kContain bool
	for i, kz := range peer.KnownZonesId {
		if kz == zoneId {
			kIndex = i
			kContain = true
			break
		}
	}
	if !kContain {
		err = fmt.Errorf("does not know this zone")
		return
	}
	if len(peer.KnownZonesId) < 2 {
		if err = zm.manager.PeerManager.DB.UpdateKnownZones(context.Background(), authorizedMember, make([]string, 0)); err != nil {
			return
		}
	} else {
		if err = zm.manager.PeerManager.DB.UpdateKnownZones(context.Background(), authorizedMember, append(peer.KnownZonesId[:kIndex], peer.KnownZonesId[kIndex+1:]...)); err != nil {
			return
		}
	}
	go signalRemovedAuthorizedMember()
	return
}
