package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

const (
	GET_PEER                    = "get_peer"
	GET_CONNECTED_PEER          = "get_connected_peer"
	LIST_PEERS                  = "list_peers"
	LIST_ALL_PEERS              = "list_all_peers"
	LIST_PEERS_BY_NAME          = "list_peers_by_name"
	LIST_PEERS_BY_ID            = "list_peers_by_id"
	UPDATE_PEER_FRIEND_REQUESTS = "update_peer_friend_requests"
	DELETE_PEER_FRIEND_REQUESTS = "delete_peer_friend_requests"
	UPDATE_PEER_FRIENDS         = "update_peer_friends"
	DELETE_PEER_FRIENDS         = "delete_peer_friends"
	CREATE_PEER                 = "create_peer"
	ADD_INCOMING_CALL           = "add_incoming_call"
	REMOVE_INCOMING_CALL        = "remove_incoming_call"
	SET_CURRENT_CALL            = "set_current_call"
)

type PeerHTTPMiddleware struct {
	manager *Manager
}

func NewPeerHTTPMiddleware(manager *Manager) *PeerHTTPMiddleware {
	return &PeerHTTPMiddleware{
		manager: manager,
	}
}

func (phm *PeerHTTPMiddleware) Process(ctx context.Context, r *ServRequest, req *http.Request, w http.ResponseWriter) (err error) {
	switch r.Type {
	case SET_CURRENT_CALL:
		if err = VerifyFields(r.Payload, "peerId", "from"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if wsPeer, ok := phm.manager.WSPeers[r.Payload["from"]]; ok {
			wsPeer.mux.Lock()
			wsPeer.CurrentCallId = r.Payload["peerId"]
			wsPeer.mux.Unlock()
		}
		err = json.NewEncoder(w).Encode(map[string]string{
			"success": "true",
		})
	case ADD_INCOMING_CALL:
		if err = VerifyFields(r.Payload, "peerId", "from"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		peerErr := phm.manager.AddIncomingCall(r.Payload["peerId"], r.Payload["from"])
		if err != nil {
			return peerErr
		}
		fmt.Println("incoming call done")
		err = json.NewEncoder(w).Encode(map[string]string{
			"success": "true",
		})
	case REMOVE_INCOMING_CALL:
		fmt.Println("removing call")
		if err = VerifyFields(r.Payload, "peerId", "from"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		peerErr := phm.manager.RemoveIncomingCall(r.Payload["peerId"], r.Payload["from"])
		if err != nil {
			return peerErr
		}
		err = json.NewEncoder(w).Encode(map[string]string{
			"success": "true",
		})
	case DELETE_PEER_FRIEND_REQUESTS:
		if err = VerifyFields(r.Payload, "peerId", "friendId"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		peerErr := phm.manager.RemovePeerFriendRequest(r.Payload["peerId"], r.Payload["friendId"])
		if err != nil {
			return peerErr
		}
		err = json.NewEncoder(w).Encode(map[string]string{
			"success": "true",
		})
		return
	case UPDATE_PEER_FRIEND_REQUESTS:
		if err = VerifyFields(r.Payload, "peerId", "friendId"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		peerErr := phm.manager.AddPeerFriendRequest(r.Payload["peerId"], r.Payload["friendId"])
		if err != nil {
			return peerErr
		}
		err = json.NewEncoder(w).Encode(map[string]string{
			"success": "true",
		})
		return
	case DELETE_PEER_FRIENDS:
		if err = VerifyFields(r.Payload, "peerId", "friendId"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		peerErr := phm.manager.RemovePeerFriend(r.Payload["peerId"], r.Payload["friendId"])
		if peerErr != nil {
			return peerErr
		}
		peerErr = phm.manager.RemovePeerFriend(r.Payload["friendId"], r.Payload["peerId"])
		if peerErr != nil {
			return peerErr
		}
		err = json.NewEncoder(w).Encode(map[string]string{
			"success": "true",
		})
		return
	case UPDATE_PEER_FRIENDS:
		if err = VerifyFields(r.Payload, "peerId", "friendId"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		peerErr := phm.manager.UpdatePeerFriends(r.Payload["peerId"], r.Payload["friendId"])
		if peerErr != nil {
			return peerErr
		}
		peerErr = phm.manager.UpdatePeerFriends(r.Payload["friendId"], r.Payload["peerId"])
		if peerErr != nil {
			return peerErr
		}
		peerErr = phm.manager.RemovePeerFriendRequest(r.Payload["peerId"], r.Payload["friendId"])
		if peerErr != nil {
			return peerErr
		}
		err = json.NewEncoder(w).Encode(map[string]string{
			"success": "true",
		})
		return
	case GET_PEER:
		if err = VerifyFields(r.Payload, "peerId"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		peer, peerErr := phm.manager.GetPeer(r.Payload["peerId"])
		if err != nil {
			return peerErr
		}
		err = json.NewEncoder(w).Encode(peer)
		return
	case GET_CONNECTED_PEER:
		if err = VerifyFields(r.Payload, "peerId"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		peer, peerErr := phm.manager.GetConnectedPeer(r.Payload["peerId"])
		if err != nil {
			return peerErr
		}
		err = json.NewEncoder(w).Encode(peer)
		return
	case LIST_ALL_PEERS:
		if err = VerifyFields(r.Payload, "lastIndex"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		peers := []*Peer{}
		for id, w := range phm.manager.WSPeers {
			fmt.Println(w.DbPeer)
			if w.DbPeer == nil {
				peers = append(peers, &Peer{
					Id:   id,
					Name: w.DisplayName,
				})
			} else {
				peers = append(peers, w.DbPeer)
			}
		}
		for id, g := range phm.manager.GRPCPeers {
			fmt.Println(g.DbPeer)
			if g.DbPeer == nil {
				peers = append(peers, &Peer{
					Id:   id,
					Name: g.DisplayName,
				})
			} else {
				peers = append(peers, g.DbPeer)
			}
		}
		err = json.NewEncoder(w).Encode(peers)
	case LIST_PEERS:
		if err = VerifyFields(r.Payload, "lastIndex"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		lastIndex, err := strconv.Atoi(r.Payload["lastIndex"])
		if err != nil {
			http.Error(w, "provide a valid integer for last index", http.StatusBadRequest)
			return err
		}
		peers, err := phm.manager.ListAllPeers(int64(lastIndex))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}
		err = json.NewEncoder(w).Encode(peers)
	case LIST_PEERS_BY_ID:
		if err = VerifyFields(r.Payload, "lastIndex", "peerId"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		lastIndex, err := strconv.Atoi(r.Payload["lastIndex"])
		if err != nil {
			http.Error(w, "provide a valid integer for last index", http.StatusBadRequest)
			return err
		}
		peers, err := phm.manager.ListPeersByID(int64(lastIndex), r.Payload["peerId"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}
		err = json.NewEncoder(w).Encode(peers)
	case LIST_PEERS_BY_NAME:
		if err = VerifyFields(r.Payload, "peerName", "lastIndex"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		lastIndex, err := strconv.Atoi(r.Payload["lastIndex"])
		if err != nil {
			http.Error(w, "provide a valid integer for last index", http.StatusBadRequest)
			return err
		}
		peers, err := phm.manager.ListPeersByName(int64(lastIndex), r.Payload["peerName"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}
		err = json.NewEncoder(w).Encode(peers)
	case CREATE_PEER:
		if err = VerifyFields(r.Payload, "peerKey", "peerId", "peerName"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err = phm.manager.CreatePeer(r.Payload["peerId"], r.Payload["peerKey"], r.Payload["peerName"]); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"peerId":  r.Payload["peerId"],
		})
	case AUTHENTICATE:
		if err = VerifyFields(r.Payload, "token", "peerId"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err = phm.manager.Authenticate(r.Payload["peerId"], r.Payload["token"]); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]string{
			"success": "true",
		})
	case PEER_AUTH_INIT:
		if err = VerifyFields(r.Payload, "peerId"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		token, tokenErr := phm.manager.PeerAuthInit(r.Payload["peerId"])
		if tokenErr != nil {
			http.Error(w, tokenErr.Error(), http.StatusInternalServerError)
			return tokenErr
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"peerId":  r.Payload["peerId"],
			"token":   token,
		})
	case PEER_AUTH_VERIFY:
		if err = VerifyFields(r.Payload, "token", "peerId"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err = phm.manager.PeerAuthVerif(r.Payload["peerId"], []byte(r.Payload["token"])); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"peerId":  r.Payload["peerId"],
			"token":   r.Payload["token"],
		})
	}
	return
}
