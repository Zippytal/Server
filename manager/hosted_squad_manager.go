package manager

import (
	"context"
	"fmt"
	"log"
	sync "sync"

	"golang.org/x/crypto/bcrypt"
)

type HostedSquadManager struct {
	DB          *HostedSquadDBManager
	manager     *Manager
	authManager *AuthManager
}

func NewHostedSquadManager(manager *Manager, authManager *AuthManager) (hostedSquadManager *HostedSquadManager, err error) {
	hostedSquadDBManager, err := NewHostedSquadDBManager("localhost", 27017)
	if err != nil {
		return
	}
	hostedSquadManager = &HostedSquadManager{
		DB:          hostedSquadDBManager,
		manager:     manager,
		authManager: authManager,
	}
	return
}

func (hsm *HostedSquadManager) GetHostedSquadByOwner(token string, owner string, lastIndex int64, networkType SquadNetworkType) (squads []*Squad, err error) {
	if _, ok := hsm.authManager.AuthTokenValid[token]; !ok {
		err = fmt.Errorf("not a valid token provided")
		return
	}
	if hsm.authManager.AuthTokenValid[token] != owner {
		err = fmt.Errorf("invalid access")
		return
	}
	fmt.Println("Net Type", networkType)
	switch networkType {
	case HOSTED:
		squads, err = hsm.DB.GetHostedSquadsByOwner(context.Background(), owner, 100, lastIndex)
		fmt.Println("squads: ", squads)
		return
	default:
		return
	}
}

func (hsm *HostedSquadManager) CreateHostedSquad(token string, id string, owner string, name string, squadType SquadType, password string, squadNetworkType SquadNetworkType, host string) (err error) {
	squadPass := ""
	if squadType == PRIVATE {
		if output, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost); err != nil {
			return err
		} else {
			squadPass = string(output)
		}
	} else {
		_ = password
	}
	squad := Squad{
		Owner:             owner,
		Name:              name,
		NetworkType:       squadNetworkType,
		HostId:            host,
		ID:                id,
		SquadType:         squadType,
		Password:          squadPass,
		Members:           make([]string, 0),
		AuthorizedMembers: make([]string, 0),
		mutex:             new(sync.RWMutex),
	}
	hsm.manager.Squads[id] = &squad
	switch squadNetworkType {

	case HOSTED:
		if err = hsm.DB.AddNewHostedSquad(context.Background(), &squad); err != nil {
			return
		}
		err = hsm.UpdateHostedSquadAuthorizedMembers(id, owner, squadNetworkType)
	}
	if err != nil {
		return
	}
	if _, ok := hsm.manager.GRPCPeers[host]; ok {
		switch squadNetworkType {
		case MESH:
			_ = hsm.manager.GRPCPeers[host].Conn.Send(&Response{
				Type:    NEW_SQUAD,
				Success: true,
				Payload: map[string]string{
					"ID": id,
				},
			})
		case HOSTED:
			_ = hsm.manager.GRPCPeers[host].Conn.Send(&Response{
				Type:    NEW_HOSTED_SQUAD,
				Success: true,
				Payload: map[string]string{
					"ID": id,
				},
			})
		}
	}
	return
}

func (hsm *HostedSquadManager) DeleteHostedSquad(token string, id string, from string, networkType SquadNetworkType) (err error) {
	switch networkType {
	case HOSTED:
		squad, squadErr := hsm.DB.GetHostedSquad(context.Background(), id)
		if squadErr != nil {
			err = fmt.Errorf("this hosted squad does not exist")
			return
		}
		for _, member := range squad.AuthorizedMembers {
			if err = hsm.DeleteHostedSquadAuthorizedMembers(id, member, squad.NetworkType); err != nil {
				return
			}
		}
		err = hsm.DB.DeleteHostedSquad(context.Background(), id)
	}
	delete(hsm.manager.Squads, id)
	return
}

func (hsm *HostedSquadManager) ModifyHostedSquad(token string, id string, from string, name string, squadType SquadType, password string) (err error) {
	squad, err := hsm.DB.GetHostedSquad(context.Background(), id)
	if squad.Owner != from {
		err = fmt.Errorf("you are not the owner of this squad so you can't modifiy it")
		return
	}
	hsm.manager.Squads[id].mutex.Lock()
	defer hsm.manager.Squads[id].mutex.Unlock()
	hsm.manager.Squads[id].Name = name
	squadPass := ""
	if squadType == PRIVATE {
		output, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		squadPass = string(output)
		hsm.manager.Squads[id].Password = squadPass
		hsm.manager.Squads[id].SquadType = PRIVATE
	} else if squadType == PUBLIC {
		hsm.manager.Squads[id].Password = squadPass
		hsm.manager.Squads[id].SquadType = PRIVATE
	}
	return
}

func (hsm *HostedSquadManager) ConnectToHostedSquad(token string, id string, from string, password string, networkType SquadNetworkType) (err error) {
	var squad *Squad

	if squad, err = hsm.DB.GetHostedSquad(context.Background(), id); err != nil {
		return
	}
	fmt.Println(token)
	fmt.Println(from)
	fmt.Println(squad.AuthorizedMembers)
	fmt.Println(hsm.authManager.AuthTokenValid[token])
	var contains bool = false
	if _, ok := hsm.authManager.AuthTokenValid[token]; ok {
		if hsm.authManager.AuthTokenValid[token] == from {
			for _, am := range squad.AuthorizedMembers {
				fmt.Println("authorized member", am)
				if am == from {
					contains = true
				}
			}
		}
	}
	fmt.Println(contains)
	var INCOMING string
	if squad.NetworkType == MESH {
		INCOMING = string(INCOMING_MEMBER)
	} else {
		INCOMING = string(HOSTED_INCOMING_MEMBER)
	}
	squad.mutex = &sync.RWMutex{}
	if squad.SquadType == PUBLIC || contains {
		squad.Join(from)
		for _, member := range squad.Members {
			if member != from {
				if _, ok := hsm.manager.GRPCPeers[member]; ok {
					if err := hsm.manager.GRPCPeers[member].Conn.Send(&Response{
						Type:    INCOMING,
						Success: true,
						Payload: map[string]string{
							"id": from,
						},
					}); err != nil {
						delete(hsm.manager.GRPCPeers, member)
						return err
					}
				} else if _, ok := hsm.manager.WSPeers[member]; ok {
					hsm.manager.WSPeers[member].mux.Lock()
					if err = hsm.manager.WSPeers[member].Conn.WriteJSON(map[string]interface{}{
						"from":    from,
						"to":      member,
						"type":    INCOMING,
						"payload": map[string]string{},
					}); err != nil {
						log.Println(err)
						hsm.manager.WSPeers[member].mux.Unlock()
						return
					}
					hsm.manager.WSPeers[member].mux.Unlock()
				}
			}
		}
		switch networkType {
		case HOSTED:
			err = hsm.DB.UpdateHostedSquadMembers(context.Background(), squad.ID, squad.Members)
			if _, ok := hsm.manager.WSPeers[from]; ok {
				hsm.manager.WSPeers[from].CurrentHostedSquadId = id
			}
		}
		return
	}
	if squad.SquadType == PRIVATE {
		if !squad.Authenticate(password) {
			err = fmt.Errorf("access denied : wrong password")
			return
		}
		squad.Join(from)
		for _, member := range squad.Members {
			if member != from {
				if _, ok := hsm.manager.GRPCPeers[member]; ok {
					if err := hsm.manager.GRPCPeers[member].Conn.Send(&Response{
						Type:    INCOMING,
						Success: true,
						Payload: map[string]string{
							"id": from,
						},
					}); err != nil {
						delete(hsm.manager.GRPCPeers, member)
						return err
					}
				} else if _, ok := hsm.manager.WSPeers[member]; ok {
					hsm.manager.WSPeers[member].mux.Lock()
					if err = hsm.manager.WSPeers[member].Conn.WriteJSON(map[string]interface{}{
						"from":    from,
						"to":      member,
						"type":    INCOMING,
						"payload": map[string]string{},
					}); err != nil {
						log.Println(err)
						hsm.manager.WSPeers[member].mux.Unlock()
						return
					}
					hsm.manager.WSPeers[member].mux.Unlock()
				}
			}
		}
		switch networkType {
		case HOSTED:
			err = hsm.DB.UpdateHostedSquadMembers(context.Background(), squad.ID, squad.Members)
			if _, ok := hsm.manager.WSPeers[from]; ok {
				hsm.manager.WSPeers[from].CurrentHostedSquadId = id
			}
		}
		return
	}
	err = fmt.Errorf("squad type is undetermined")
	return
}

func (hsm *HostedSquadManager) LeaveHostedSquad(id string, from string, networkType SquadNetworkType) (err error) {
	var squad *Squad
	if squad, err = hsm.DB.GetHostedSquad(context.Background(), id); err != nil {
		return
	}
	squad.mutex = &sync.RWMutex{}
	var memberIndex int
	for i, member := range squad.Members {
		if member == from {
			memberIndex = i
			break
		}
	}
	squad.mutex.Lock()
	fmt.Println(squad.Members)
	if len(squad.Members) < 2 {
		newMembers := []string{}
		squad.Members = newMembers
	} else {
		squad.Members[len(squad.Members)-1], squad.Members[memberIndex] = squad.Members[memberIndex], squad.Members[len(squad.Members)-1]
		newMembers := squad.Members[:len(squad.Members)-1]
		squad.Members = newMembers
	}
	squad.mutex.Unlock()
	// manager.RLock()
	// defer manager.RUnlock()
	var LEAVING string

	LEAVING = string(HOSTED_LEAVING_MEMBER)
	for _, member := range squad.Members {
		if member != from {
			if _, ok := hsm.manager.GRPCPeers[member]; ok {
				if wserr := hsm.manager.GRPCPeers[member].Conn.Send(&Response{
					Type:    "hosted_squad_stop_call",
					Success: true,
					Payload: map[string]string{
						"from":    from,
						"squadId": id,
					},
				}); wserr != nil {
					delete(hsm.manager.GRPCPeers, member)
					continue
				}
			} else if _, ok := hsm.manager.WSPeers[member]; ok {
				hsm.manager.WSPeers[member].mux.Lock()
				if err = hsm.manager.WSPeers[member].Conn.WriteJSON(map[string]interface{}{
					"from":    from,
					"to":      member,
					"type":    LEAVING,
					"payload": map[string]string{},
				}); err != nil {
					log.Println(err)
					hsm.manager.WSPeers[member].mux.Unlock()
					continue
				}
				hsm.manager.WSPeers[member].mux.Unlock()
			}
		}
	}
	for _, member := range squad.AuthorizedMembers {
		if member != from {
			if _, ok := hsm.manager.GRPCPeers[member]; ok {
				if err := hsm.manager.GRPCPeers[member].Conn.Send(&Response{
					Type:    "hosted_squad_stop_call",
					Success: true,
					Payload: map[string]string{
						"from":    from,
						"squadId": id,
					},
				}); err != nil {
					delete(hsm.manager.GRPCPeers, member)
					continue
				}
			} else if _, ok := hsm.manager.WSPeers[member]; ok {
				hsm.manager.WSPeers[member].mux.Lock()
				if err = hsm.manager.WSPeers[member].Conn.WriteJSON(map[string]interface{}{
					"from":    from,
					"to":      member,
					"type":    LEAVING,
					"payload": map[string]string{},
				}); err != nil {
					log.Println(err)
					hsm.manager.WSPeers[member].mux.Unlock()
					continue
				}
				hsm.manager.WSPeers[member].mux.Unlock()
			}
		}
	}
	fmt.Println("done", squad.Members)
	switch networkType {
	case HOSTED:
		err = hsm.DB.UpdateHostedSquadMembers(context.Background(), squad.ID, squad.Members)
		if _, ok := hsm.manager.WSPeers[from]; ok {
			hsm.manager.WSPeers[from].CurrentSquadId = ""
		}
	}
	return
}

func (hsm *HostedSquadManager) ListAllHostedSquads(lastIndex int64, networkType SquadNetworkType) (squads []*Squad, err error) {
	switch networkType {
	case HOSTED:
		squads, err = hsm.DB.GetHostedSquads(context.Background(), 100, lastIndex)
	}
	return
}

func (hsm *HostedSquadManager) ListHostedSquadsByName(lastIndex int64, squadName string, networkType SquadNetworkType) (squads []*Squad, err error) {
	switch networkType {
	case HOSTED:
		squads, err = hsm.DB.GetHostedSquadsByName(context.Background(), squadName, 100, lastIndex)
	}
	return
}

func (hsm *HostedSquadManager) ListHostedSquadsByHost(lastIndex int64, host string, networkType SquadNetworkType) (squads []*Squad, err error) {
	switch networkType {
	case HOSTED:
		squads, err = hsm.DB.GetHostedSquadsByHost(context.Background(), host, 100, lastIndex)
	}
	return
}

func (hsm *HostedSquadManager) ListHostedSquadsByID(lastIndex int64, squadId string, networkType SquadNetworkType) (squads []*Squad, err error) {
	switch networkType {
	case HOSTED:
		squads, err = hsm.DB.GetHostedSquadsByID(context.Background(), squadId, 100, lastIndex)
	}
	return
}

func (hsm *HostedSquadManager) GetHostedSquadByID(squadId string, networkType SquadNetworkType) (squads *Squad, err error) {
	switch networkType {
	case HOSTED:
		squads, err = hsm.DB.GetHostedSquadByID(context.Background(), squadId)
	}
	return
}

func (hsm *HostedSquadManager) UpdateHostedSquadName(squadId string, squadName string, networkType SquadNetworkType) (err error) {
	switch networkType {
	case HOSTED:
		err = hsm.DB.UpdateHostedSquadName(context.Background(), squadId, squadName)
	}
	return
}

func (hsm *HostedSquadManager) UpdateHostedSquadAuthorizedMembers(squadId string, authorizedMembers string, networkType SquadNetworkType) (err error) {
	wg, errCh, done := &sync.WaitGroup{}, make(chan error), make(chan struct{})
	wg.Add(2)
	go func() {
		defer wg.Done()
		var squad *Squad
		switch networkType {
		case HOSTED:
			if squad, err = hsm.DB.GetHostedSquad(context.Background(), squadId); err != nil {
				errCh <- err
				return
			}
		}
		for _, v := range squad.AuthorizedMembers {
			if v == authorizedMembers {
				err = fmt.Errorf("user already authorized")
				errCh <- err
				return
			}
		}
		switch networkType {
		case HOSTED:
			if err = hsm.DB.UpdateHostedSquadAuthorizedMembers(context.Background(), squadId, append(squad.AuthorizedMembers, authorizedMembers)); err != nil {
				errCh <- err
				return
			}
			if _, ok := hsm.manager.GRPCPeers[authorizedMembers]; ok {
				if err := hsm.manager.GRPCPeers[authorizedMembers].Conn.Send(&Response{
					Type:    NEW_AUTHORIZED_HOSTED_SQUAD,
					Success: true,
					Payload: map[string]string{
						"id": squadId,
					},
				}); err != nil {
					delete(hsm.manager.GRPCPeers, authorizedMembers)
					return
				}
			} else if _, ok := hsm.manager.WSPeers[authorizedMembers]; ok {
				hsm.manager.WSPeers[authorizedMembers].mux.Lock()
				if err = hsm.manager.WSPeers[authorizedMembers].Conn.WriteJSON(map[string]interface{}{
					"from": "server",
					"to":   authorizedMembers,
					"type": NEW_AUTHORIZED_HOSTED_SQUAD,
					"payload": map[string]string{
						"squadId": squadId,
					},
				}); err != nil {
					log.Println(err)
					hsm.manager.WSPeers[authorizedMembers].mux.Unlock()
					return
				}
				hsm.manager.WSPeers[authorizedMembers].mux.Unlock()
			}
		}
	}()
	go func() {
		defer wg.Done()
		var peer *Peer
		if peer, err = hsm.manager.PeerManager.GetPeer(authorizedMembers); err != nil {
			errCh <- err
			return
		}
		switch networkType {
		case HOSTED:
			for _, v := range peer.KnownHostedSquadsId {
				if v == squadId {
					err = fmt.Errorf("hosted squad already known")
					errCh <- err
					return
				}
			}
			if err = hsm.manager.PeerManager.DB.UpdateKnownHostedSquads(context.Background(), authorizedMembers, append(peer.KnownHostedSquadsId, squadId)); err != nil {
				errCh <- err
				return
			}
		}
	}()
	go func() {
		wg.Wait()
		done <- struct{}{}
	}()
	select {
	case err = <-errCh:
		return
	case <-done:
		return
	}
}

func (hsm *HostedSquadManager) DeleteHostedSquadAuthorizedMembers(squadId string, authorizedMembers string, networkType SquadNetworkType) (err error) {
	wg, errCh, done := &sync.WaitGroup{}, make(chan error), make(chan struct{})
	wg.Add(2)
	go func() {
		defer wg.Done()
		var squad *Squad

		if squad, err = hsm.DB.GetHostedSquad(context.Background(), squadId); err != nil {
			errCh <- err
			return
		}
		var index int
		var contain bool
		for i, v := range squad.AuthorizedMembers {
			if v == authorizedMembers {
				index = i
				contain = true
				break
			}
		}
		if !contain {
			err = fmt.Errorf("member not in squad")
			return
		}
		if len(squad.AuthorizedMembers) < 2 {
			if err = hsm.DB.UpdateHostedSquadAuthorizedMembers(context.Background(), squadId, make([]string, 0)); err != nil {
				errCh <- err
				return
			}
		} else {
			if err = hsm.DB.UpdateHostedSquadAuthorizedMembers(context.Background(), squadId, append(squad.AuthorizedMembers[:index], squad.AuthorizedMembers[index+1:]...)); err != nil {
				errCh <- err
				return
			}
		}
		if _, ok := hsm.manager.GRPCPeers[authorizedMembers]; ok {
			if err := hsm.manager.GRPCPeers[authorizedMembers].Conn.Send(&Response{
				Type:    REMOVED_AUTHORIZED_HOSTED_SQUAD,
				Success: true,
				Payload: map[string]string{
					"id": squadId,
				},
			}); err != nil {
				delete(hsm.manager.GRPCPeers, authorizedMembers)
				return
			}
		} else if _, ok := hsm.manager.WSPeers[authorizedMembers]; ok {
			fmt.Println("--------------------- sending event to", authorizedMembers, "-------------------")
			hsm.manager.WSPeers[authorizedMembers].mux.Lock()
			if err = hsm.manager.WSPeers[authorizedMembers].Conn.WriteJSON(map[string]interface{}{
				"from": "server",
				"to":   authorizedMembers,
				"type": REMOVED_AUTHORIZED_HOSTED_SQUAD,
				"payload": map[string]string{
					"squadId": squadId,
				},
			}); err != nil {
				log.Println(err)
				hsm.manager.WSPeers[authorizedMembers].mux.Unlock()
				return
			}
			hsm.manager.WSPeers[authorizedMembers].mux.Unlock()
		} else {
			fmt.Println("------------------- no corresponding peer with id ----------------")
		}
	}()
	go func() {
		defer wg.Done()
		var peer *Peer
		if peer, err = hsm.manager.PeerManager.GetPeer(authorizedMembers); err != nil {
			errCh <- err
			return
		}
		var index int
		for i, v := range peer.KnownHostedSquadsId {
			if v == squadId {
				index = i
			}
		}
		if len(peer.KnownHostedSquadsId) < 2 {
			peer.KnownHostedSquadsId = make([]string, 0)
			if err = hsm.manager.PeerManager.DB.UpdateKnownHostedSquads(context.Background(), authorizedMembers, peer.KnownHostedSquadsId); err != nil {
				errCh <- err
				return
			}
		} else {
			peer.KnownHostedSquadsId[len(peer.KnownHostedSquadsId)-1], peer.KnownHostedSquadsId[index] = peer.KnownHostedSquadsId[index], peer.KnownHostedSquadsId[len(peer.KnownHostedSquadsId)-1]
			if err = hsm.manager.PeerManager.DB.UpdateKnownHostedSquads(context.Background(), authorizedMembers, peer.KnownHostedSquadsId[:len(peer.KnownHostedSquadsId)-1]); err != nil {
				errCh <- err
				return
			}
		}
	}()
	go func() {
		wg.Wait()
		done <- struct{}{}
	}()
	select {
	case err = <-errCh:
		return
	case <-done:
		return
	}
}

func (hsm *HostedSquadManager) UpdateHostedSquadPassword(squadId string, password string) (err error) {
	pass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return
	}
	err = hsm.DB.UpdateHostedSquadName(context.Background(), squadId, string(pass))
	return
}
