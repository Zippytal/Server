package manager

import (
	"context"
	"fmt"
	"log"
	sync "sync"

	"golang.org/x/crypto/bcrypt"
)

type SquadManager struct {
	DB          *SquadDBManager
	manager     *Manager
	authManager *AuthManager
}

func NewSquadManager(manager *Manager, authManager *AuthManager) (squadManager *SquadManager, err error) {
	squadDBManager, err := NewSquadDBManager("localhost", 27017)
	if err != nil {
		return
	}
	squadManager = &SquadManager{
		DB:          squadDBManager,
		manager:     manager,
		authManager: authManager,
	}
	return
}

func (sm *SquadManager) GetSquadSByOwner(token string, owner string, lastIndex int64, networkType SquadNetworkType) (squads []*Squad, err error) {
	if _, ok := sm.authManager.AuthTokenValid[token]; !ok {
		err = fmt.Errorf("not a valid token provided")
		return
	}
	if sm.authManager.AuthTokenValid[token] != owner {
		err = fmt.Errorf("invalid access")
		return
	}
	fmt.Println("Net Type", networkType)
	switch networkType {
	case MESH:
		squads, err = sm.DB.GetSquadsByOwner(context.Background(), owner, 100, lastIndex)
		fmt.Println("squads: ", squads)
		return
	default:
		return
	}
}

func (sm *SquadManager) CreateSquad(token string, id string, owner string, name string, squadType SquadType, password string, squadNetworkType SquadNetworkType, host string) (err error) {
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
	sm.manager.Squads[id] = &squad
	switch squadNetworkType {
	case MESH:
		if err = sm.DB.AddNewSquad(context.Background(), &squad); err != nil {
			return
		}
		err = sm.UpdateSquadAuthorizedMembers(id, owner, squadNetworkType)
	}
	if err != nil {
		return
	}
	if _, ok := sm.manager.GRPCPeers[host]; ok {
		switch squadNetworkType {
		case MESH:
			_ = sm.manager.GRPCPeers[host].Conn.Send(&Response{
				Type:    NEW_SQUAD,
				Success: true,
				Payload: map[string]string{
					"ID": id,
				},
			})
		}
	}
	return
}

func (sm *SquadManager) DeleteSquad(token string, id string, from string, networkType SquadNetworkType) (err error) {
	switch networkType {
	case MESH:
		squad, squadErr := sm.DB.GetSquad(context.Background(), id)
		if squadErr != nil {
			err = fmt.Errorf("this squad does not exist")
			return
		}
		for _, member := range squad.AuthorizedMembers {
			if err = sm.DeleteSquadAuthorizedMembers(id, member, squad.NetworkType); err != nil {
				return
			}
		}
		err = sm.DB.DeleteSquad(context.Background(), id)
	}
	delete(sm.manager.Squads, id)
	return
}

func (sm *SquadManager) ModifySquad(token string, id string, from string, name string, squadType SquadType, password string) (err error) {
	squad, err := sm.DB.GetSquad(context.Background(), id)
	if squad.Owner != from {
		err = fmt.Errorf("you are not the owner of this squad so you can't modifiy it")
		return
	}
	sm.manager.Squads[id].mutex.Lock()
	defer sm.manager.Squads[id].mutex.Unlock()
	sm.manager.Squads[id].Name = name
	squadPass := ""
	if squadType == PRIVATE {
		output, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		squadPass = string(output)
		sm.manager.Squads[id].Password = squadPass
		sm.manager.Squads[id].SquadType = PRIVATE
	} else if squadType == PUBLIC {
		sm.manager.Squads[id].Password = squadPass
		sm.manager.Squads[id].SquadType = PRIVATE
	}
	return
}

func (sm *SquadManager) ConnectToSquad(token string, id string, from string, password string, networkType SquadNetworkType) (err error) {
	var squad *Squad
	if squad, err = sm.DB.GetSquad(context.Background(), id); err != nil {
		return
	}
	fmt.Println(token)
	fmt.Println(from)
	fmt.Println(squad.AuthorizedMembers)
	fmt.Println(sm.authManager.AuthTokenValid[token])
	var contains bool = false
	if _, ok := sm.authManager.AuthTokenValid[token]; ok {
		if sm.authManager.AuthTokenValid[token] == from {
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
	}
	squad.mutex = &sync.RWMutex{}
	if squad.SquadType == PUBLIC || contains {
		squad.Join(from)
		for _, member := range squad.Members {
			if member != from {
				if _, ok := sm.manager.GRPCPeers[member]; ok {
					if err := sm.manager.GRPCPeers[member].Conn.Send(&Response{
						Type:    INCOMING,
						Success: true,
						Payload: map[string]string{
							"id": from,
						},
					}); err != nil {
						delete(sm.manager.GRPCPeers, member)
						return err
					}
				} else if _, ok := sm.manager.WSPeers[member]; ok {
					sm.manager.WSPeers[member].mux.Lock()
					if err = sm.manager.WSPeers[member].Conn.WriteJSON(map[string]interface{}{
						"from":    from,
						"to":      member,
						"type":    INCOMING,
						"payload": map[string]string{},
					}); err != nil {
						log.Println(err)
						sm.manager.WSPeers[member].mux.Unlock()
						return
					}
					sm.manager.WSPeers[member].mux.Unlock()
				}
			}
		}
		switch networkType {
		case MESH:
			err = sm.DB.UpdateSquadMembers(context.Background(), squad.ID, squad.Members)
			if _, ok := sm.manager.WSPeers[from]; ok {
				sm.manager.WSPeers[from].CurrentSquadId = id
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
				if _, ok := sm.manager.GRPCPeers[member]; ok {
					if err := sm.manager.GRPCPeers[member].Conn.Send(&Response{
						Type:    INCOMING,
						Success: true,
						Payload: map[string]string{
							"id": from,
						},
					}); err != nil {
						delete(sm.manager.GRPCPeers, member)
						return err
					}
				} else if _, ok := sm.manager.WSPeers[member]; ok {
					sm.manager.WSPeers[member].mux.Lock()
					if err = sm.manager.WSPeers[member].Conn.WriteJSON(map[string]interface{}{
						"from":    from,
						"to":      member,
						"type":    INCOMING,
						"payload": map[string]string{},
					}); err != nil {
						log.Println(err)
						sm.manager.WSPeers[member].mux.Unlock()
						return
					}
					sm.manager.WSPeers[member].mux.Unlock()
				}
			}
		}
		switch networkType {
		case MESH:
			err = sm.DB.UpdateSquadMembers(context.Background(), squad.ID, squad.Members)
			if _, ok := sm.manager.WSPeers[from]; ok {
				sm.manager.WSPeers[from].CurrentSquadId = id
			}

		}
		return
	}
	err = fmt.Errorf("squad type is undetermined")
	return
}

func (sm *SquadManager) LeaveSquad(id string, from string, networkType SquadNetworkType) (err error) {
	var squad *Squad
	if networkType == HOSTED {
		fmt.Println("---------------  wtf ------------")
		return
	}
	if networkType == MESH {
		if squad, err = sm.DB.GetSquad(context.Background(), id); err != nil {
			return
		}
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
	if squad.NetworkType == MESH {
		LEAVING = string(LEAVING_MEMBER)
	}
	for _, member := range squad.Members {
		if member != from {
			if _, ok := sm.manager.GRPCPeers[member]; ok {
				if wserr := sm.manager.GRPCPeers[member].Conn.Send(&Response{
					Type:    "hosted_squad_stop_call",
					Success: true,
					Payload: map[string]string{
						"from":    from,
						"squadId": id,
					},
				}); wserr != nil {
					delete(sm.manager.GRPCPeers, member)
					continue
				}
			} else if _, ok := sm.manager.WSPeers[member]; ok {
				sm.manager.WSPeers[member].mux.Lock()
				if err = sm.manager.WSPeers[member].Conn.WriteJSON(map[string]interface{}{
					"from":    from,
					"to":      member,
					"type":    LEAVING,
					"payload": map[string]string{},
				}); err != nil {
					log.Println(err)
					sm.manager.WSPeers[member].mux.Unlock()
					continue
				}
				sm.manager.WSPeers[member].mux.Unlock()
			}
		}
	}
	for _, member := range squad.AuthorizedMembers {
		if member != from {
			if _, ok := sm.manager.GRPCPeers[member]; ok {
				if err := sm.manager.GRPCPeers[member].Conn.Send(&Response{
					Type:    "hosted_squad_stop_call",
					Success: true,
					Payload: map[string]string{
						"from":    from,
						"squadId": id,
					},
				}); err != nil {
					delete(sm.manager.GRPCPeers, member)
					continue
				}
			} else if _, ok := sm.manager.WSPeers[member]; ok {
				sm.manager.WSPeers[member].mux.Lock()
				if err = sm.manager.WSPeers[member].Conn.WriteJSON(map[string]interface{}{
					"from":    from,
					"to":      member,
					"type":    LEAVING,
					"payload": map[string]string{},
				}); err != nil {
					log.Println(err)
					sm.manager.WSPeers[member].mux.Unlock()
					continue
				}
				sm.manager.WSPeers[member].mux.Unlock()
			}
		}
	}
	fmt.Println("done", squad.Members)
	switch networkType {
	case MESH:
		err = sm.DB.UpdateSquadMembers(context.Background(), squad.ID, squad.Members)
		if _, ok := sm.manager.WSPeers[from]; ok {
			sm.manager.WSPeers[from].CurrentSquadId = ""
		}
	}
	return
}

func (sm *SquadManager) ListAllSquads(lastIndex int64, networkType SquadNetworkType) (squads []*Squad, err error) {
	switch networkType {
	case MESH:
		squads, err = sm.DB.GetSquads(context.Background(), 100, lastIndex)
	}
	return
}

func (sm *SquadManager) ListSquadsByName(lastIndex int64, squadName string, networkType SquadNetworkType) (squads []*Squad, err error) {
	switch networkType {
	case MESH:
		squads, err = sm.DB.GetSquadsByName(context.Background(), squadName, 100, lastIndex)
	}
	return
}

func (sm *SquadManager) ListSquadsByID(lastIndex int64, squadId string, networkType SquadNetworkType) (squads []*Squad, err error) {
	switch networkType {
	case MESH:
		squads, err = sm.DB.GetSquadsByID(context.Background(), squadId, 100, lastIndex)
	}
	return
}

func (sm *SquadManager) GetSquadByID(squadId string, networkType SquadNetworkType) (squads *Squad, err error) {
	switch networkType {
	case MESH:
		squads, err = sm.DB.GetSquadByID(context.Background(), squadId)
	}
	return
}

func (sm *SquadManager) UpdateSquadName(squadId string, squadName string, networkType SquadNetworkType) (err error) {
	switch networkType {
	case MESH:
		err = sm.DB.UpdateSquadName(context.Background(), squadId, squadName)
	}
	return
}

func (sm *SquadManager) UpdateSquadAuthorizedMembers(squadId string, authorizedMembers string, networkType SquadNetworkType) (err error) {
	wg, errCh, done := &sync.WaitGroup{}, make(chan error), make(chan struct{})
	wg.Add(2)
	go func() {
		defer wg.Done()
		var squad *Squad
		switch networkType {
		case HOSTED:
			fmt.Println("---------------- wtf --------------")
		case MESH:
			if squad, err = sm.DB.GetSquad(context.Background(), squadId); err != nil {
				errCh <- err
				return
			}
			for _, v := range squad.AuthorizedMembers {
				if v == authorizedMembers {
					err = fmt.Errorf("user already authorized")
					errCh <- err
					return
				}
			}
		}
		switch networkType {
		case MESH:
			if err = sm.DB.UpdateSquadAuthorizedMembers(context.Background(), squadId, append(squad.AuthorizedMembers, authorizedMembers)); err != nil {
				errCh <- err
				return
			}
			if _, ok := sm.manager.GRPCPeers[authorizedMembers]; ok {
				if err := sm.manager.GRPCPeers[authorizedMembers].Conn.Send(&Response{
					Type:    NEW_AUTHORIZED_SQUAD,
					Success: true,
					Payload: map[string]string{
						"id": squadId,
					},
				}); err != nil {
					delete(sm.manager.GRPCPeers, authorizedMembers)
					return
				}
			} else if _, ok := sm.manager.WSPeers[authorizedMembers]; ok {
				sm.manager.WSPeers[authorizedMembers].mux.Lock()
				if err = sm.manager.WSPeers[authorizedMembers].Conn.WriteJSON(map[string]interface{}{
					"from": "server",
					"to":   authorizedMembers,
					"type": NEW_AUTHORIZED_SQUAD,
					"payload": map[string]string{
						"squadId": squadId,
					},
				}); err != nil {
					log.Println(err)
					sm.manager.WSPeers[authorizedMembers].mux.Unlock()
					return
				}
				sm.manager.WSPeers[authorizedMembers].mux.Unlock()
			}
		}
	}()
	go func() {
		defer wg.Done()
		var peer *Peer
		if peer, err = sm.manager.PeerManager.GetPeer(authorizedMembers); err != nil {
			errCh <- err
			return
		}
		switch networkType {
		case MESH:
			for _, v := range peer.KnownSquadsId {
				if v == squadId {
					err = fmt.Errorf("squad already known")
					errCh <- err
					return
				}
			}
			if err = sm.manager.PeerManager.DB.UpdateKnownSquads(context.Background(), authorizedMembers, append(peer.KnownSquadsId, squadId)); err != nil {
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

func (sm *SquadManager) DeleteSquadAuthorizedMembers(squadId string, authorizedMembers string, networkType SquadNetworkType) (err error) {
	wg, errCh, done := &sync.WaitGroup{}, make(chan error), make(chan struct{})
	wg.Add(2)
	go func() {
		defer wg.Done()
		var squad *Squad
		switch networkType {
		case MESH:
			if squad, err = sm.DB.GetSquad(context.Background(), squadId); err != nil {
				errCh <- err
				return
			}
		}
		var index int
		var contain bool
		for i, v := range squad.AuthorizedMembers {
			if v == authorizedMembers {
				index = i
				contain = true
			}
		}
		if !contain {
			err = fmt.Errorf("member not in squad")
			return
		}
		switch networkType {
		case MESH:
			if len(squad.AuthorizedMembers) < 2 {
				if err = sm.DB.UpdateSquadAuthorizedMembers(context.Background(), squadId, make([]string, 0)); err != nil {
					errCh <- err
					return
				}
			} else {
				if err = sm.DB.UpdateSquadAuthorizedMembers(context.Background(), squadId, append(squad.AuthorizedMembers[:index], squad.AuthorizedMembers[index+1:]...)); err != nil {
					errCh <- err
					return
				}
			}
			if _, ok := sm.manager.GRPCPeers[authorizedMembers]; ok {
				if err := sm.manager.GRPCPeers[authorizedMembers].Conn.Send(&Response{
					Type:    REMOVED_AUTHORIZED_SQUAD,
					Success: true,
					Payload: map[string]string{
						"id": squadId,
					},
				}); err != nil {
					delete(sm.manager.GRPCPeers, authorizedMembers)
					return
				}
			} else if _, ok := sm.manager.WSPeers[authorizedMembers]; ok {
				sm.manager.WSPeers[authorizedMembers].mux.Lock()
				if err = sm.manager.WSPeers[authorizedMembers].Conn.WriteJSON(map[string]interface{}{
					"from": "server",
					"to":   authorizedMembers,
					"type": REMOVED_AUTHORIZED_SQUAD,
					"payload": map[string]string{
						"squadId": squadId,
					},
				}); err != nil {
					log.Println(err)
					sm.manager.WSPeers[authorizedMembers].mux.Unlock()
					return
				}
				sm.manager.WSPeers[authorizedMembers].mux.Unlock()
			}
		}
	}()
	go func() {
		defer wg.Done()
		var peer *Peer
		if peer, err = sm.manager.PeerManager.GetPeer(authorizedMembers); err != nil {
			errCh <- err
			return
		}
		switch networkType {
		case MESH:
			var index int
			for i, v := range peer.KnownSquadsId {
				if v == squadId {
					index = i
				}
			}
			if len(peer.KnownHostedSquadsId) < 2 {
				peer.KnownHostedSquadsId = make([]string, 0)
				if err = sm.manager.PeerManager.DB.UpdateKnownSquads(context.Background(), authorizedMembers, peer.KnownHostedSquadsId); err != nil {
					errCh <- err
					return
				}
			} else {
				peer.KnownSquadsId[len(peer.KnownSquadsId)-1], peer.KnownSquadsId[index] = peer.KnownSquadsId[index], peer.KnownSquadsId[len(peer.KnownSquadsId)-1]
				if err = sm.manager.PeerManager.DB.UpdateKnownSquads(context.Background(), authorizedMembers, peer.KnownSquadsId[:len(peer.KnownSquadsId)-1]); err != nil {
					errCh <- err
					return
				}
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

func (sm *SquadManager) UpdateSquadPassword(squadId string, password string) (err error) {
	pass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return
	}
	err = sm.DB.UpdateSquadName(context.Background(), squadId, string(pass))
	return
}
