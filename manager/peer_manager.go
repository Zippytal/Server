package manager

import (
	"context"
	"fmt"
	"log"
)

type PeerManager struct {
	DB          *PeerDBManager
	manager     *Manager
	authManager *AuthManager
}

func NewPeerManager(manager *Manager, authManager *AuthManager) (peerManager *PeerManager, err error) {
	peerDBManager, err := NewPeerDBManager("localhost", 27017)
	if err != nil {
		return
	}
	peerManager = &PeerManager{
		DB:          peerDBManager,
		manager:     manager,
		authManager: authManager,
	}
	return
}

func (pm *PeerManager) RemovePeerFriendRequest(peerId string, friendId string) (err error) {
	peer, err := pm.DB.GetPeer(context.Background(), peerId)
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
		err = pm.DB.UpdatePeerFriendRequests(context.Background(), peerId, append(peer.FriendRequests[:index], peer.FriendRequests[index+1:]...))
	}
	return
}

func (pm *PeerManager) AddPeerFriendRequest(peerId string, friendId string) (err error) {
	peer, err := pm.DB.GetPeer(context.Background(), peerId)
	if err != nil {
		return
	}
	for _, v := range peer.FriendRequests {
		if v == friendId {
			err = fmt.Errorf("request already sent")
			return
		}
	}
	err = pm.DB.UpdatePeerFriendRequests(context.Background(), peerId, append(peer.FriendRequests, friendId))
	if _, ok := pm.manager.GRPCPeers[peerId]; ok {
		if err := pm.manager.GRPCPeers[peerId].Conn.Send(&Response{
			Type:    NEW_FRIEND_REQUEST,
			Success: true,
			Payload: map[string]string{
				"id": friendId,
			},
		}); err != nil {
			delete(pm.manager.GRPCPeers, peerId)
			return err
		}
	} else if _, ok := pm.manager.WSPeers[peerId]; ok {
		pm.manager.WSPeers[peerId].mux.Lock()
		if err = pm.manager.WSPeers[peerId].Conn.WriteJSON(map[string]interface{}{
			"from":    friendId,
			"to":      peerId,
			"type":    NEW_FRIEND_REQUEST,
			"payload": map[string]string{},
		}); err != nil {
			log.Println(err)
			pm.manager.WSPeers[peerId].mux.Unlock()
			return
		}
		pm.manager.WSPeers[peerId].mux.Unlock()
	}
	return
}

func (pm *PeerManager) GetPeer(peerId string) (peer *Peer, err error) {
	peer, err = pm.DB.GetPeer(context.Background(), peerId)
	return
}

func (pm *PeerManager) AddIncomingCall(peerId string, from string) (err error) {
	fmt.Println("adding incoming call")
	peer, err := pm.DB.GetPeer(context.Background(), peerId)
	if err != nil {
		return
	}
	incomingCalls := append(peer.IncomingCalls, from)
	if err = pm.DB.UpdateIncomingCalls(context.Background(), peerId, incomingCalls); err != nil {
		return
	}
	if _, ok := pm.manager.WSPeers[peerId]; ok {
		pm.manager.WSPeers[peerId].CurrentCallId = from
	}
	return
}

func (pm *PeerManager) RemoveIncomingCall(peerId string, from string) (err error) {
	peer, err := pm.DB.GetPeer(context.Background(), peerId)
	if err != nil {
		return
	}
	var incomingCalls []string
	if len(peer.IncomingCalls) < 2 {
		var found bool
		for _, p := range peer.IncomingCalls {
			if p == from {
				found = true
				break
			}
		}
		if found {
			incomingCalls = make([]string, 0)
		} else {
			incomingCalls = peer.IncomingCalls
		}
	} else {
		for i, p := range peer.IncomingCalls {
			if p == from {
				incomingCalls = append(peer.IncomingCalls[:i], peer.IncomingCalls[i+1:]...)
			}
		}
	}
	err = pm.DB.UpdateIncomingCalls(context.Background(), peerId, incomingCalls)
	if _, ok := pm.manager.WSPeers[from]; ok {
		pm.manager.WSPeers[from].CurrentCallId = ""
	}
	return
}

func (pm *PeerManager) GetConnectedPeer(peerId string) (peer *Peer, err error) {
	if _, ok := pm.manager.WSPeers[peerId]; !ok {
		err = fmt.Errorf("no peer found")
		return
	}
	peer = &Peer{
		Id:   peerId,
		Name: pm.manager.WSPeers[peerId].DisplayName,
	}
	return
}

func (pm *PeerManager) RemovePeerFriend(peerId string, friendId string) (err error) {
	peer, err := pm.DB.GetPeer(context.Background(), peerId)
	if err != nil {
		return
	}
	var index = 0
	for i, v := range peer.FriendRequests {
		if v == friendId {
			index = i
		}
	}
	if err = pm.DB.UpdatePeerFriendRequests(context.Background(), peerId, append(peer.FriendRequests[:index], peer.FriendRequests[index+1:]...)); err != nil {
		return
	}
	friendPeer, err := pm.DB.GetPeer(context.Background(), friendId)
	if err != nil {
		return
	}
	var x = 0
	for i, v := range peer.FriendRequests {
		if v == friendId {
			x = i
		}
	}
	if err = pm.DB.UpdatePeerFriendRequests(context.Background(), friendId, append(friendPeer.FriendRequests[:x], friendPeer.FriendRequests[x+1:]...)); err != nil {
		return
	}
	if _, ok := pm.manager.GRPCPeers[peerId]; ok {
		if err := pm.manager.GRPCPeers[peerId].Conn.Send(&Response{
			Type:    DELETE_FRIEND,
			Success: true,
			Payload: map[string]string{
				"id": friendId,
			},
		}); err != nil {
			delete(pm.manager.GRPCPeers, peerId)
			return err
		}
	} else if _, ok := pm.manager.WSPeers[peerId]; ok {
		pm.manager.WSPeers[peerId].mux.Lock()
		if err = pm.manager.WSPeers[peerId].Conn.WriteJSON(map[string]interface{}{
			"from":    peerId,
			"to":      friendId,
			"type":    DELETE_FRIEND,
			"payload": map[string]string{},
		}); err != nil {
			log.Println(err)
			pm.manager.WSPeers[peerId].mux.Unlock()
			return
		}
		pm.manager.WSPeers[peerId].mux.Unlock()
	}
	return
}

func (pm *PeerManager) UpdatePeerFriends(peerId string, friendId string) (err error) {
	peer, err := pm.DB.GetPeer(context.Background(), peerId)
	if err != nil {
		return
	}
	err = pm.RemovePeerFriendRequest(peerId, friendId)
	if err != nil {
		return
	}
	err = pm.DB.UpdatePeerFriends(context.Background(), peerId, append(peer.Friends, friendId))
	return
}

func (pm *PeerManager) CreatePeer(peerId string, peerKey string, peerUsername string) (err error) {
	peer := &Peer{
		PubKey:              peerKey,
		Id:                  peerId,
		Name:                peerUsername,
		IncomingCalls:       []string{},
		Friends:             []string{},
		KnownSquadsId:       []string{},
		KnownHostedSquadsId: []string{},
		FriendRequests:      []string{},
		Active:              true,
	}
	err = pm.DB.AddNewPeer(context.Background(), peer)
	return
}

func (pm *PeerManager) PeerAuthInit(peerId string) (encryptedToken []byte, err error) {
	if _, ok := pm.authManager.AuthTokenPending[peerId]; ok {
		err = fmt.Errorf("user in authentification")
		return
	}
	peer, err := pm.DB.GetPeer(context.Background(), peerId)
	if err != nil {
		delete(pm.authManager.AuthTokenPending, peerId)
		return
	}
	encryptedToken, err = pm.authManager.GenerateAuthToken(peer.Id, peer.PubKey)
	if err != nil {
		delete(pm.authManager.AuthTokenPending, peerId)
	}
	return
}

func (pm *PeerManager) PeerAuthVerif(peerId string, token []byte) (err error) {
	if _, ok := pm.authManager.AuthTokenPending[peerId]; !ok {
		err = fmt.Errorf("the peer %s have not initiated auth", peerId)
		return
	}
	if pm.authManager.AuthTokenPending[peerId] != string(token) {
		err = fmt.Errorf("authentification failed wrong key")
	} else {
		pm.authManager.AuthTokenValid[string(token)] = peerId
	}
	fmt.Println("done")
	delete(pm.authManager.AuthTokenPending, peerId)
	return
}

func (pm *PeerManager) Authenticate(peerId string, token string) (err error) {
	fmt.Println(pm.authManager)
	if _, ok := pm.authManager.AuthTokenValid[token]; !ok {
		err = fmt.Errorf("authentification failed token invalid")
		return
	}
	if pm.authManager.AuthTokenValid[token] != peerId {
		err = fmt.Errorf("authentification failed token associated with wrong id")
	}
	return
}

func (pm *PeerManager) ListAllPeers(lastIndex int64) (peers []*Peer, err error) {
	peers, err = pm.DB.GetPeers(context.Background(), 100, lastIndex)
	return
}

func (pm *PeerManager) ListPeersByID(lastIndex int64, id string) (peers []*Peer, err error) {

	peers, err = pm.DB.GetPeersByID(context.Background(), id, 100, lastIndex)
	return
}

func (pm *PeerManager) ListPeersByName(lastIndex int64, name string) (peers []*Peer, err error) {

	peers, err = pm.DB.GetPeersByName(context.Background(), name, 100, lastIndex)
	return
}
