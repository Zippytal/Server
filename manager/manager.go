package manager

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"
)

type (
	ManagerState  uint8
	GRPCPeerState uint8
	WSState       uint8
	SquadType     string
	SquadEvent    string

	GRPCPeer struct {
		Conn        GrpcManager_LinkServer
		State       GRPCPeerState
		DisplayName string
		DbPeer      *Peer
	}

	WSPeer struct {
		Conn        *websocket.Conn
		DisplayName string
		DbPeer      *Peer
		State       WSState
	}

	Manager struct {
		State                ManagerState
		GRPCPeers            map[string]*GRPCPeer
		WSPeers              map[string]*WSPeer
		Squads               map[string]*Squad
		SquadDBManager       *SquadDBManager
		HostedSquadDBManager *HostedSquadDBManager
		PeerDBManager        *PeerDBManager
		AuthManager          *AuthManager
		NodeDBManager        *NodeDBManager
		*sync.RWMutex
	}
)

const (
	PRIVATE SquadType = "private"
	PUBLIC  SquadType = "public"
)

const (
	INCOMING_MEMBER        SquadEvent = "incoming_member"
	HOSTED_INCOMING_MEMBER SquadEvent = "hosted_incoming_member"
	LEAVING_MEMBER         SquadEvent = "leaving_member"
	HOSTED_LEAVING_MEMBER  SquadEvent = "hosted_leaving_member"
	NEW_HOSTED_SQUAD                  = "new_hosted_squad"
)

const (
	ON ManagerState = iota
	OFF
)

const (
	CONNECTED GRPCPeerState = iota
	SLEEP
)

const DB_NAME string = "zippytal_server"

func NewManager() (manager *Manager, err error) {
	hostedSquadDBManager, err := NewHostedSquadDBManager("localhost", 27017)
	if err != nil {
		return
	}
	squadDBManager, err := NewSquadDBManager("localhost", 27017)
	if err != nil {
		return
	}
	peerDBManager, err := NewPeerDBManager("localhost", 27017)
	if err != nil {
		return
	}
	nodeDBManager, err := NewNodeDBManager("localhost", 27017)
	if err != nil {
		return
	}
	manager = &Manager{
		State:                ON,
		GRPCPeers:            make(map[string]*GRPCPeer),
		WSPeers:              make(map[string]*WSPeer),
		Squads:               make(map[string]*Squad),
		SquadDBManager:       squadDBManager,
		HostedSquadDBManager: hostedSquadDBManager,
		PeerDBManager:        peerDBManager,
		NodeDBManager:        nodeDBManager,
		RWMutex:              &sync.RWMutex{},
		AuthManager:          NewAuthManager(),
	}
	return
}

func (manager *Manager) RemovePeerFriendRequest(peerId string, friendId string) (err error) {
	peer, err := manager.PeerDBManager.GetPeer(context.Background(), peerId)
	if err != nil {
		return
	}
	var index = 0
	for i, v := range peer.FriendRequests {
		if v == friendId {
			index = i
		}
	}
	if len(peer.FriendRequests) > 0 {
		err = manager.PeerDBManager.UpdatePeerFriendRequests(context.Background(), peerId, append(peer.FriendRequests[:index], peer.FriendRequests[index+1:]...))
	}
	return
}

func (manager *Manager) AddPeerFriendRequest(peerId string, friendId string) (err error) {
	peer, err := manager.PeerDBManager.GetPeer(context.Background(), peerId)
	if err != nil {
		return
	}
	for _, v := range peer.FriendRequests {
		if v == friendId {
			err = fmt.Errorf("request already sent")
			return
		}
	}
	err = manager.PeerDBManager.UpdatePeerFriendRequests(context.Background(), peerId, append(peer.FriendRequests, friendId))
	return
}

func (manager *Manager) GetPeer(peerId string) (peer *Peer, err error) {
	peer, err = manager.PeerDBManager.GetPeer(context.Background(), peerId)
	return
}

func (manager *Manager) RemovePeerFriend(peerId string, friendId string) (err error) {
	peer, err := manager.PeerDBManager.GetPeer(context.Background(), peerId)
	if err != nil {
		return
	}
	var index = 0
	for i, v := range peer.FriendRequests {
		if v == friendId {
			index = i
		}
	}
	err = manager.PeerDBManager.UpdatePeerFriendRequests(context.Background(), peerId, append(peer.FriendRequests[:index], peer.FriendRequests[index+1:]...))
	return
}

func (manager *Manager) UpdatePeerFriends(peerId string, friendId string) (err error) {
	peer, err := manager.PeerDBManager.GetPeer(context.Background(), peerId)
	if err != nil {
		return
	}
	err = manager.RemovePeerFriendRequest(peerId, friendId)
	if err != nil {
		return
	}
	err = manager.PeerDBManager.UpdatePeerFriends(context.Background(), peerId, append(peer.Friends, friendId))
	return
}

func (manager *Manager) CreatePeer(peerId string, peerKey string, peerUsername string) (err error) {
	peer := &Peer{
		PubKey:         peerKey,
		Id:             peerId,
		Name:           peerUsername,
		Friends:        []string{},
		KnownSquadsId:  []string{},
		FriendRequests: []string{},
		Active:         true,
	}
	err = manager.PeerDBManager.AddNewPeer(context.Background(), peer)
	return
}

func (manager *Manager) CreateNode(nodeId string, nodeKey string, nodeUsername string) (err error) {
	node := &Node{
		PubKey:         nodeKey,
		Id:             nodeId,
		Name:           nodeUsername,
		Friends:        []string{},
		KnownSquadsId:  []string{},
		FriendRequests: []string{},
		Active:         true,
	}
	err = manager.NodeDBManager.AddNewNode(context.Background(), node)
	return
}

func (manager *Manager) PeerAuthInit(peerId string) (encryptedToken []byte, err error) {
	if _, ok := manager.AuthManager.AuthTokenPending[peerId]; ok {
		err = fmt.Errorf("user in authentification")
		return
	}
	peer, err := manager.PeerDBManager.GetPeer(context.Background(), peerId)
	if err != nil {
		delete(manager.AuthManager.AuthTokenPending, peerId)
		return
	}
	encryptedToken, err = manager.AuthManager.GenerateAuthToken(peer.Id, peer.PubKey)
	if err != nil {
		delete(manager.AuthManager.AuthTokenPending, peerId)
	}
	return
}

func (manager *Manager) PeerAuthVerif(peerId string, token []byte) (err error) {
	if _, ok := manager.AuthManager.AuthTokenPending[peerId]; !ok {
		err = fmt.Errorf("the peer %s have not initiated auth", peerId)
		return
	}
	if manager.AuthManager.AuthTokenPending[peerId] != string(token) {
		err = fmt.Errorf("authentification failed wrong key")
	} else {
		manager.AuthManager.AuthTokenValid[string(token)] = peerId
	}
	fmt.Println("done")
	delete(manager.AuthManager.AuthTokenPending, peerId)
	return
}

func (manager *Manager) Authenticate(peerId string, token string) (err error) {
	if _, ok := manager.AuthManager.AuthTokenValid[token]; !ok {
		err = fmt.Errorf("authentification failed token invalid")
		return
	}
	if manager.AuthManager.AuthTokenValid[token] != peerId {
		err = fmt.Errorf("authentification failed token associated with wrong id")
	}
	return
}

func (manager *Manager) GetSquadSByOwner(token string, owner string, lastIndex int64, networkType SquadNetworkType) (squads []*Squad, err error) {
	if _, ok := manager.AuthManager.AuthTokenValid[token]; !ok {
		err = fmt.Errorf("not a valid token provided")
		return
	}
	if manager.AuthManager.AuthTokenValid[token] != owner {
		err = fmt.Errorf("invalid access")
		return
	}
	fmt.Println("Net Type", networkType)
	switch networkType {
	case MESH:
		squads, err = manager.SquadDBManager.GetSquadsByOwner(context.Background(), owner, 100, lastIndex)
		fmt.Println("squads: ", squads)
		return
	case HOSTED:
		squads, err = manager.HostedSquadDBManager.GetHostedSquadsByOwner(context.Background(), owner, 100, lastIndex)
		fmt.Println("squads: ", squads)
		return
	default:
		return
	}
}

func (manager *Manager) CreateSquad(token string, id string, owner string, name string, squadType SquadType, password string, squadNetworkType SquadNetworkType, host string) (err error) {
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
	manager.Squads[id] = &squad
	switch squadNetworkType {
	case MESH:
		err = manager.SquadDBManager.AddNewSquad(context.Background(), &squad)
	case HOSTED:
		err = manager.HostedSquadDBManager.AddNewHostedSquad(context.Background(), &squad)
	}
	if err != nil {
		return
	}
	if _, ok := manager.GRPCPeers[host]; ok {
		_ = manager.GRPCPeers[host].Conn.Send(&Response{
			Type:    NEW_HOSTED_SQUAD,
			Success: true,
			Payload: map[string]string{
				"ID": id,
			},
		})
	}
	return
}

func (manager *Manager) DeleteSquad(token string, id string, from string, networkType SquadNetworkType) (err error) {
	switch networkType {
	case MESH:
		err = manager.SquadDBManager.DeleteSquad(context.Background(), id)
	case HOSTED:
		err = manager.HostedSquadDBManager.DeleteHostedSquad(context.Background(), id)
	}
	delete(manager.Squads, id)
	return
}

func (manager *Manager) ModifySquad(token string, id string, from string, name string, squadType SquadType, password string) (err error) {
	squad, err := manager.SquadDBManager.GetSquad(context.Background(), id)
	if squad.Owner != from {
		err = fmt.Errorf("you are not the owner of this squad so you can't modifiy it")
		return
	}
	manager.Squads[id].mutex.Lock()
	defer manager.Squads[id].mutex.Unlock()
	manager.Squads[id].Name = name
	squadPass := ""
	if squadType == PRIVATE {
		output, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		squadPass = string(output)
		manager.Squads[id].Password = squadPass
		manager.Squads[id].SquadType = PRIVATE
	} else if squadType == PUBLIC {
		manager.Squads[id].Password = squadPass
		manager.Squads[id].SquadType = PRIVATE
	}
	return
}

func (manager *Manager) ConnectToSquad(token string, id string, from string, password string, networkType SquadNetworkType) (err error) {
	var squad *Squad
	if networkType == MESH {
		if squad, err = manager.SquadDBManager.GetSquad(context.Background(), id); err != nil {
			return
		}
	} else if networkType == HOSTED {
		if squad, err = manager.HostedSquadDBManager.GetHostedSquad(context.Background(), id); err != nil {
			return
		}
	}
	fmt.Println(token)
	fmt.Println(from)
	fmt.Println(squad.AuthorizedMembers)
	fmt.Println(manager.AuthManager.AuthTokenValid[token])
	var contains bool = false
	if _, ok := manager.AuthManager.AuthTokenValid[token]; ok {
		if manager.AuthManager.AuthTokenValid[token] == from {
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
				if _, ok := manager.GRPCPeers[member]; ok {
					if err := manager.GRPCPeers[member].Conn.Send(&Response{
						Type:    INCOMING,
						Success: true,
						Payload: map[string]string{
							"id": from,
						},
					}); err != nil {
						delete(manager.GRPCPeers, member)
						return err
					}
				} else if _, ok := manager.WSPeers[member]; ok {
					if err = manager.WSPeers[member].Conn.WriteJSON(map[string]interface{}{
						"from":    from,
						"to":      member,
						"type":    INCOMING,
						"payload": map[string]string{},
					}); err != nil {
						log.Println(err)
						return
					}
				}
			}
		}
		switch networkType {
		case MESH:
			err = manager.SquadDBManager.UpdateSquadMembers(context.Background(), squad.ID, squad.Members)
		case HOSTED:
			err = manager.HostedSquadDBManager.UpdateHostedSquadMembers(context.Background(), squad.ID, squad.Members)
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
				if _, ok := manager.GRPCPeers[member]; ok {
					if err := manager.GRPCPeers[member].Conn.Send(&Response{
						Type:    INCOMING,
						Success: true,
						Payload: map[string]string{
							"id": from,
						},
					}); err != nil {
						delete(manager.GRPCPeers, member)
						return err
					}
				} else if _, ok := manager.WSPeers[member]; ok {
					if err = manager.WSPeers[member].Conn.WriteJSON(map[string]interface{}{
						"from":    from,
						"to":      member,
						"type":    INCOMING,
						"payload": map[string]string{},
					}); err != nil {
						log.Println(err)
						return
					}
				}
			}
		}
		switch networkType {
		case MESH:
			err = manager.SquadDBManager.UpdateSquadMembers(context.Background(), squad.ID, squad.Members)
		case HOSTED:
			err = manager.HostedSquadDBManager.UpdateHostedSquadMembers(context.Background(), squad.ID, squad.Members)
		}
		return
	}
	err = fmt.Errorf("squad type is undetermined")
	return
}

func (manager *Manager) LeaveSquad(id string, from string, networkType SquadNetworkType) (err error) {
	var squad *Squad
	if networkType == MESH {
		if squad, err = manager.SquadDBManager.GetSquad(context.Background(), id); err != nil {
			return
		}
	} else if networkType == HOSTED {
		if squad, err = manager.HostedSquadDBManager.GetHostedSquad(context.Background(), id); err != nil {
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
	manager.RLock()
	var LEAVING string
	if squad.NetworkType == MESH {
		LEAVING = string(LEAVING_MEMBER)
	} else {
		LEAVING = string(HOSTED_LEAVING_MEMBER)
	}
	defer manager.RUnlock()
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		for _, member := range squad.Members {
			if member != from {
				if _, ok := manager.GRPCPeers[member]; ok {
					if err := manager.GRPCPeers[member].Conn.Send(&Response{
						Type:    LEAVING,
						Success: true,
						Payload: map[string]string{
							"id":      from,
							"squadId": id,
						},
					}); err != nil {
						delete(manager.GRPCPeers, member)
						return
					}
				} else if _, ok := manager.WSPeers[member]; ok {
					if err = manager.WSPeers[member].Conn.WriteJSON(map[string]interface{}{
						"from":    from,
						"to":      member,
						"type":    LEAVING,
						"payload": map[string]string{},
					}); err != nil {
						log.Println(err)
						return
					}
				}
			}
		}
	}()
	go func() {
		defer wg.Done()
		for _, member := range squad.AuthorizedMembers {
			if member != from {
				if _, ok := manager.GRPCPeers[member]; ok {
					if err := manager.GRPCPeers[member].Conn.Send(&Response{
						Type:    LEAVING,
						Success: true,
						Payload: map[string]string{
							"id":      from,
							"squadId": id,
						},
					}); err != nil {
						delete(manager.GRPCPeers, member)
						return
					}
				} else if _, ok := manager.WSPeers[member]; ok {
					if err = manager.WSPeers[member].Conn.WriteJSON(map[string]interface{}{
						"from":    from,
						"to":      member,
						"type":    LEAVING,
						"payload": map[string]string{},
					}); err != nil {
						log.Println(err)
						return
					}
				}
			}
		}
	}()
	fmt.Println(squad.Members)
	switch networkType {
	case MESH:
		err = manager.SquadDBManager.UpdateSquadMembers(context.Background(), squad.ID, squad.Members)
	case HOSTED:
		err = manager.HostedSquadDBManager.UpdateHostedSquadMembers(context.Background(), squad.ID, squad.Members)
	}
	wg.Wait()
	return
}

func (manager *Manager) ListAllSquads(lastIndex int64, networkType SquadNetworkType) (squads []*Squad, err error) {
	switch networkType {
	case MESH:
		squads, err = manager.SquadDBManager.GetSquads(context.Background(), 100, lastIndex)
	case HOSTED:
		squads, err = manager.HostedSquadDBManager.GetHostedSquads(context.Background(), 100, lastIndex)
	}
	return
}

func (manager *Manager) ListSquadsByName(lastIndex int64, squadName string, networkType SquadNetworkType) (squads []*Squad, err error) {
	switch networkType {
	case MESH:
		squads, err = manager.SquadDBManager.GetSquadsByName(context.Background(), squadName, 100, lastIndex)
	case HOSTED:
		squads, err = manager.HostedSquadDBManager.GetHostedSquadsByName(context.Background(), squadName, 100, lastIndex)
	}
	return
}

func (manager *Manager) ListSquadsByHost(lastIndex int64, host string, networkType SquadNetworkType) (squads []*Squad, err error) {
	switch networkType {
	case HOSTED:
		squads, err = manager.HostedSquadDBManager.GetHostedSquadsByHost(context.Background(), host, 100, lastIndex)
	}
	return
}

func (manager *Manager) ListSquadsByID(lastIndex int64, squadId string, networkType SquadNetworkType) (squads []*Squad, err error) {
	switch networkType {
	case MESH:
		squads, err = manager.SquadDBManager.GetSquadsByID(context.Background(), squadId, 100, lastIndex)
	case HOSTED:
		squads, err = manager.HostedSquadDBManager.GetHostedSquadsByID(context.Background(), squadId, 100, lastIndex)
	}
	return
}

func (manager *Manager) ListAllPeers(lastIndex int64) (peers []*Peer, err error) {
	peers, err = manager.PeerDBManager.GetPeers(context.Background(), 100, lastIndex)
	return
}

func (manager *Manager) ListPeersByID(lastIndex int64, id string) (peers []*Peer, err error) {

	peers, err = manager.PeerDBManager.GetPeersByID(context.Background(), id, 100, lastIndex)
	return
}

func (manager *Manager) ListPeersByName(lastIndex int64, name string) (peers []*Peer, err error) {

	peers, err = manager.PeerDBManager.GetPeersByName(context.Background(), name, 100, lastIndex)
	return
}

func (manager *Manager) UpdateSquadName(squadId string, squadName string, networkType SquadNetworkType) (err error) {
	switch networkType {
	case MESH:
		err = manager.SquadDBManager.UpdateSquadName(context.Background(), squadId, squadName)
	case HOSTED:
		err = manager.HostedSquadDBManager.UpdateHostedSquadName(context.Background(), squadId, squadName)
	}
	return
}

func (manager *Manager) UpdateSquadAuthorizedMembers(squadId string, authorizedMembers string, networkType SquadNetworkType) (err error) {
	wg, errCh, done := &sync.WaitGroup{}, make(chan error), make(chan struct{})
	wg.Add(2)
	go func() {
		defer wg.Done()
		var squad *Squad
		switch networkType {
		case MESH:
			if squad, err = manager.SquadDBManager.GetSquad(context.Background(), squadId); err != nil {
				errCh <- err
				return
			}
		case HOSTED:
			if squad, err = manager.HostedSquadDBManager.GetHostedSquad(context.Background(), squadId); err != nil {
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
		case MESH:
			if err = manager.SquadDBManager.UpdateSquadAuthorizedMembers(context.Background(), squadId, append(squad.AuthorizedMembers, authorizedMembers)); err != nil {
				errCh <- err
				return
			}
		case HOSTED:
			if err = manager.HostedSquadDBManager.UpdateHostedSquadAuthorizedMembers(context.Background(), squadId, append(squad.AuthorizedMembers, authorizedMembers)); err != nil {
				errCh <- err
				return
			}
		}
	}()
	go func() {
		defer wg.Done()
		var peer *Peer
		if peer, err = manager.PeerDBManager.GetPeer(context.Background(), authorizedMembers); err != nil {
			errCh <- err
			return
		}
		for _, v := range peer.KnownSquadsId {
			if v == squadId {
				err = fmt.Errorf("squad already known")
				errCh <- err
				return
			}
		}
		if err = manager.PeerDBManager.UpdateKnownSquads(context.Background(),authorizedMembers,append(peer.KnownSquadsId,squadId)); err != nil {
			errCh <- err
			return
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

func (manager *Manager) DeleteSquadAuthorizedMembers(squadId string, authorizedMembers string, networkType SquadNetworkType) (err error) {
	var squad *Squad
	switch networkType {
	case MESH:
		if squad, err = manager.SquadDBManager.GetSquad(context.Background(), squadId); err != nil {
			return
		}
	case HOSTED:
		if squad, err = manager.HostedSquadDBManager.GetHostedSquad(context.Background(), squadId); err != nil {
			return
		}
	}
	var index int
	for i, v := range squad.AuthorizedMembers {
		if v == authorizedMembers {
			index = i
		}
	}
	squad.AuthorizedMembers[len(squad.AuthorizedMembers)-1], squad.AuthorizedMembers[index] = squad.AuthorizedMembers[index], squad.AuthorizedMembers[len(squad.AuthorizedMembers)-1]
	switch networkType {
	case MESH:
		err = manager.SquadDBManager.UpdateSquadAuthorizedMembers(context.Background(), squadId, squad.AuthorizedMembers[:len(squad.AuthorizedMembers)-1])
	case HOSTED:
		err = manager.HostedSquadDBManager.UpdateHostedSquadAuthorizedMembers(context.Background(), squadId, squad.AuthorizedMembers[:len(squad.AuthorizedMembers)-1])
	}
	err = manager.SquadDBManager.UpdateSquadAuthorizedMembers(context.Background(), squadId, squad.AuthorizedMembers[:len(squad.AuthorizedMembers)-1])
	return
}

func (manager *Manager) UpdateSquadPassword(squadId string, password string) (err error) {
	pass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return
	}
	err = manager.SquadDBManager.UpdateSquadName(context.Background(), squadId, string(pass))
	return
}

func (manager *Manager) AddGrpcPeer(peer GrpcManager_LinkServer, id string, req *Request) (err error) {
	fmt.Printf("adding peer %s\n", req.From)
	manager.Lock()
	manager.GRPCPeers[req.From] = &GRPCPeer{Conn: peer, State: CONNECTED}
	manager.Unlock()
	if _, ok := req.Payload["to"]; ok {
		if _, ok := manager.GRPCPeers[req.From]; ok {
			if err = manager.GRPCPeers[req.From].Conn.Send(&Response{
				Type:    req.Type,
				Success: true,
				Payload: req.Payload,
			}); err != nil {
				return
			}
		}
	}
	err = manager.manage(peer)
	delete(manager.GRPCPeers, req.From)
	return
}

func (manager *Manager) manage(peer GrpcManager_LinkServer) (err error) {
	done, errch := make(chan struct{}), make(chan error)
	go func() {
		for {
			req, err := peer.Recv()
			if err != nil {
				errch <- err
				return
			}
			fmt.Println(req)
			if _, ok := req.Payload["to"]; ok {
				to := req.Payload["to"]
				if _, ok := manager.GRPCPeers[to]; ok {
					if err := manager.GRPCPeers[to].Conn.Send(&Response{
						Type:    req.Type,
						Success: true,
						Payload: req.Payload,
					}); err != nil {
						errch <- err
						return
					}
				} else if _, ok := manager.WSPeers[to]; ok {
					if err = manager.WSPeers[to].Conn.WriteJSON(map[string]interface{}{
						"from":    req.From,
						"to":      to,
						"type":    req.Type,
						"payload": req.Payload,
					}); err != nil {
						log.Println(err)
						return
					}
				}
			}
		}
	}()
	select {
	case <-done:
		log.Println("manage is done")
		return
	case err = <-errch:
		return
	case <-peer.Context().Done():
		err = peer.Context().Err()
		return
	}
}
