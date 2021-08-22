package manager

import (
	"encoding/json"
	"fmt"
	"log"
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

type PeerHTTPMiddleware struct{}

func (shm *PeerHTTPMiddleware) Process(r *ServRequest, req *http.Request, w http.ResponseWriter, m *Manager) (err error) {
	fmt.Println("got req in peer http middleware of type : ", r.Type)
	switch r.Type {
	case SET_CURRENT_CALL:
		if err = VerifyFields(r.Payload, "peerId", "from"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if wsPeer, ok := m.WSPeers[r.Payload["from"]]; ok {
			wsPeer.mux.Lock()
			wsPeer.CurrentCallId = r.Payload["peerId"]
			wsPeer.mux.Unlock()
		}
		err = json.NewEncoder(w).Encode(map[string]string{
			"success": "true",
		})
	case ADD_INCOMING_CALL:
		fmt.Println("incoming call")
		if err = VerifyFields(r.Payload, "peerId", "from"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		peerErr := m.AddIncomingCall(r.Payload["peerId"], r.Payload["from"])
		if err != nil {
			log.Println("an error occured in add incoming call", peerErr)
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
		peerErr := m.RemoveIncomingCall(r.Payload["peerId"], r.Payload["from"])
		if err != nil {
			log.Println("an error occured in add incoming call", peerErr)
			return peerErr
		}
		fmt.Println("incoming call done")
		err = json.NewEncoder(w).Encode(map[string]string{
			"success": "true",
		})
	case DELETE_PEER_FRIEND_REQUESTS:
		if err = VerifyFields(r.Payload, "peerId", "friendId"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		peerErr := m.RemovePeerFriendRequest(r.Payload["peerId"], r.Payload["friendId"])
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
		peerErr := m.AddPeerFriendRequest(r.Payload["peerId"], r.Payload["friendId"])
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
		peerErr := m.RemovePeerFriend(r.Payload["peerId"], r.Payload["friendId"])
		if peerErr != nil {
			return peerErr
		}
		peerErr = m.RemovePeerFriend(r.Payload["friendId"], r.Payload["peerId"])
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
		peerErr := m.UpdatePeerFriends(r.Payload["peerId"], r.Payload["friendId"])
		if peerErr != nil {
			return peerErr
		}
		peerErr = m.UpdatePeerFriends(r.Payload["friendId"], r.Payload["peerId"])
		if peerErr != nil {
			return peerErr
		}
		peerErr = m.RemovePeerFriendRequest(r.Payload["peerId"], r.Payload["friendId"])
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
		peer, peerErr := m.GetPeer(r.Payload["peerId"])
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
		peer, peerErr := m.GetConnectedPeer(r.Payload["peerId"])
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
		for id, w := range m.WSPeers {
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
		for id, g := range m.GRPCPeers {
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
		peers, err := m.ListAllPeers(int64(lastIndex))
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
		peers, err := m.ListPeersByID(int64(lastIndex), r.Payload["peerId"])
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
		peers, err := m.ListPeersByName(int64(lastIndex), r.Payload["peerName"])
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
		if err = m.CreatePeer(r.Payload["peerId"], r.Payload["peerKey"], r.Payload["peerName"]); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"peerId":  r.Payload["peerId"],
		})
	}
	return
}
