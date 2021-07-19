package manager

import (
	"encoding/json"
	"net/http"
	"strconv"
)

const (
	LIST_PEER                       = "list_peer"
	JOIN_SQUAD                      = "join_squad"
	LIST_SQUADS                     = "list_squads"
	LIST_SQUADS_BY_NAME             = "list_squads_by_name"
	LIST_SQUADS_BY_ID               = "list_squads_by_id"
	LIST_PEERS                      = "list_peers"
	LIST_PEERS_BY_NAME              = "list_peers_by_name"
	LIST_PEERS_BY_ID                = "list_peers_by_id"
	GET_SQUADS_BY_OWNER             = "get_squads_by_owner"
	SQUAD_ACCESS_DENIED             = "squad_access_denied"
	SQUAD_ACCESS_GRANTED            = "squad_access_granted"
	LEAVE_SQUAD                     = "leave_squad"
	SQUAD_AUTH                      = "auth_squad"
	CREATE_SQUAD                    = "create_squad"
	DELETE_SQUAD                    = "delete_squad"
	MODIFY_SQUAD                    = "modify_squad"
	UPDATE_SQUAD_NAME               = "update_squad_name"
	UPDATE_SQUAD_AUTHORIZED_MEMBERS = "update_squad_authorized_members"
	UPDATE_SQUAD_PASSWORD           = "update_squad_password"
	PEER_AUTH_INIT                  = "peer_auth_init"
	PEER_AUTH_VERIFY                = "peer_auth_verify"
	CREATE_PEER                     = "create_peer"
)

type SquadHTTPMiddleware struct{}

func (shm *SquadHTTPMiddleware) Process(r *ServRequest, req *http.Request, w http.ResponseWriter, m *Manager) (err error) {
	switch r.Type {
	case LIST_PEERS:
		if _, ok := r.Payload["lastIndex"]; !ok {
			http.Error(w, "no field lastIndex in payload", http.StatusBadRequest)
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
		if _, ok := r.Payload["peerId"]; !ok {
			http.Error(w, "no field peerId in payload", http.StatusBadRequest)
			return
		}
		if _, ok := r.Payload["lastIndex"]; !ok {
			http.Error(w, "no field lastIndex in payload", http.StatusBadRequest)
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
		if _, ok := r.Payload["peerName"]; !ok {
			http.Error(w, "no field peerName in payload", http.StatusBadRequest)
			return
		}
		if _, ok := r.Payload["lastIndex"]; !ok {
			http.Error(w, "no field lastIndex in payload", http.StatusBadRequest)
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
	case GET_SQUADS_BY_OWNER:
		if _, ok := r.Payload["owner"]; !ok {
			http.Error(w, "no field owner in payload", http.StatusBadRequest)
			return
		}
		if _, ok := r.Payload["lastIndex"]; !ok {
			http.Error(w, "no field lastIndex in payload", http.StatusBadRequest)
			return
		}
		lastIndex, err := strconv.Atoi(r.Payload["lastIndex"])
		if err != nil {
			http.Error(w, "field lastIndex is not an int", http.StatusBadRequest)
			return err
		}
		squads, err := m.GetSquadSByOwner(r.Token, r.Payload["owner"], int64(lastIndex))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return err
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"squads":  squads,
		})
	case CREATE_PEER:
		if _, ok := r.Payload["peerId"]; !ok {
			http.Error(w, "no field peerId in payload", http.StatusBadRequest)
			return
		}
		if _, ok := r.Payload["peerKey"]; !ok {
			http.Error(w, "no field peerKey in payload", http.StatusBadRequest)
			return
		}
		if _, ok := r.Payload["peerName"]; !ok {
			http.Error(w, "no field peerName in payload", http.StatusBadRequest)
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
	case PEER_AUTH_INIT:
		if _, ok := r.Payload["peerId"]; !ok {
			http.Error(w, "no field peerId in payload", http.StatusBadRequest)
			return
		}
		token, err := m.PeerAuthInit(r.Payload["peerId"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"peerId":  r.Payload["peerId"],
			"token":   token,
		})
	case PEER_AUTH_VERIFY:
		if _, ok := r.Payload["peerId"]; !ok {
			http.Error(w, "no field peerId in payload", http.StatusBadRequest)
			return
		}
		if _, ok := r.Payload["token"]; !ok {
			http.Error(w, "no field peerKey in payload", http.StatusBadRequest)
			return
		}
		if err = m.PeerAuthVerif(r.Payload["peerId"], []byte(r.Payload["token"])); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"peerId":  r.Payload["peerId"],
			"token":   r.Payload["token"],
		})
	case LIST_PEER:
		peers := []*Peer{}
		for id := range m.GRPCPeers {
			peers = append(peers, &Peer{
				Id:   id,
				Name: "unknow",
			})
		}
		for id := range m.WSPeers {
			peers = append(peers, &Peer{
				Id:   id,
				Name: "unknow",
			})
		}
		err = json.NewEncoder(w).Encode(peers)
		return
	case JOIN_SQUAD:
		if _, ok := r.Payload["squadId"]; !ok {
			http.Error(w, "no field squadId in payload", http.StatusBadRequest)
			return
		}
		if _, ok := r.Payload["password"]; !ok {
			http.Error(w, "no field password in payload", http.StatusBadRequest)
			return
		}
		if _, ok := r.Payload["networkType"]; !ok {
			http.Error(w, "no field networkType in payload", http.StatusBadRequest)
			return
		}
		if err = m.ConnectToSquad(r.Token, r.Payload["squadId"], r.From, r.Payload["password"], r.Payload["networkType"]); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"squadId": r.Payload["squadId"],
		})
	case LIST_SQUADS:
		if _, ok := r.Payload["networkType"]; !ok {
			http.Error(w, "no field networkType in payload", http.StatusBadRequest)
			return
		}
		if _, ok := r.Payload["lastIndex"]; !ok {
			http.Error(w, "no field lastIndex in payload", http.StatusBadRequest)
			return
		}
		lastIndex, err := strconv.Atoi(r.Payload["lastIndex"])
		if err != nil {
			http.Error(w, "provide a valid integer for last index", http.StatusBadRequest)
			return err
		}
		squads, err := m.ListAllSquads(int64(lastIndex), r.Payload["networkType"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}
		err = json.NewEncoder(w).Encode(squads)
	case LIST_SQUADS_BY_NAME:
		if _, ok := r.Payload["squadName"]; !ok {
			http.Error(w, "no field squadName in payload", http.StatusBadRequest)
			return
		}
		if _, ok := r.Payload["networkType"]; !ok {
			http.Error(w, "no field networkType in payload", http.StatusBadRequest)
			return
		}
		if _, ok := r.Payload["lastIndex"]; !ok {
			http.Error(w, "no field lastIndex in payload", http.StatusBadRequest)
			return
		}
		lastIndex, err := strconv.Atoi(r.Payload["lastIndex"])
		if err != nil {
			http.Error(w, "provide a valid integer for last index", http.StatusBadRequest)
			return err
		}
		squads, err := m.ListSquadsByName(int64(lastIndex), r.Payload["squadName"], r.Payload["networkType"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}
		err = json.NewEncoder(w).Encode(squads)
	case LIST_SQUADS_BY_ID:
		if _, ok := r.Payload["squadId"]; !ok {
			http.Error(w, "no field squadId in payload", http.StatusBadRequest)
			return
		}
		if _, ok := r.Payload["networkType"]; !ok {
			http.Error(w, "no field networkType in payload", http.StatusBadRequest)
			return
		}
		if _, ok := r.Payload["lastIndex"]; !ok {
			http.Error(w, "no field lastIndex in payload", http.StatusBadRequest)
			return
		}
		lastIndex, err := strconv.Atoi(r.Payload["lastIndex"])
		if err != nil {
			http.Error(w, "provide a valid integer for last index", http.StatusBadRequest)
			return err
		}
		squads, err := m.ListSquadsByID(int64(lastIndex), r.Payload["squadId"], r.Payload["networkType"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}
		err = json.NewEncoder(w).Encode(squads)
	case LEAVE_SQUAD:
		if _, ok := r.Payload["squadId"]; !ok {
			http.Error(w, "no field squadId in payload", http.StatusBadRequest)
			return
		}
		if _, ok := r.Payload["squadNetworkType"]; !ok {
			http.Error(w, "no field squadNetworkType in payload", http.StatusBadRequest)
			return
		}
		if err = m.LeaveSquad(r.Payload["squadId"], r.From, r.Payload["squadNetworkType"]); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"squadId": r.Payload["squadId"],
		})
	case SQUAD_AUTH:
	case CREATE_SQUAD:
		if _, ok := r.Payload["squadId"]; !ok {
			http.Error(w, "no field squadId in payload", http.StatusBadRequest)
			return
		}
		if _, ok := r.Payload["password"]; !ok {
			http.Error(w, "no field password in payload", http.StatusBadRequest)
			return
		}
		if _, ok := r.Payload["squadType"]; !ok {
			http.Error(w, "no field squadType in payload", http.StatusBadRequest)
			return
		}
		if _, ok := r.Payload["squadName"]; !ok {
			http.Error(w, "no field squadName in payload", http.StatusBadRequest)
			return
		}
		if _, ok := r.Payload["squadNetworkType"]; !ok {
			http.Error(w, "no field squadNetworkType in payload", http.StatusBadRequest)
			return
		}
		if r.Payload["squadNetworkType"] == HOSTED {
			if _, ok := r.Payload["squadHost"]; !ok {
				http.Error(w, "no field squadHost in payload", http.StatusBadRequest)
				return
			}
		}
		if err = m.CreateSquad(r.Token, r.Payload["squadId"], r.From, r.Payload["squadName"], SquadType(r.Payload["squadType"]), r.Payload["password"], r.Payload["squadNetworkType"], r.Payload["squadHost"]); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	case DELETE_SQUAD:
		if _, ok := r.Payload["squadId"]; !ok {
			http.Error(w, "no field squadId in payload", http.StatusBadRequest)
			return
		}
		if err = m.DeleteSquad(r.Token, r.Payload["squadId"], r.From); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	case MODIFY_SQUAD:
		if _, ok := r.Payload["squadId"]; !ok {
			http.Error(w, "no field squadId in payload", http.StatusBadRequest)
			return
		}
		if _, ok := r.Payload["password"]; !ok {
			http.Error(w, "no field password in payload", http.StatusBadRequest)
			return
		}
		if _, ok := r.Payload["squadName"]; !ok {
			http.Error(w, "no field squadName in payload", http.StatusBadRequest)
			return
		}
		if _, ok := r.Payload["squadType"]; !ok {
			http.Error(w, "no field squadType in payload", http.StatusBadRequest)
			return
		}
		if err = m.ModifySquad(r.Token, r.Payload["squadId"], r.From, r.Payload["squadName"], SquadType(r.Payload["squadType"]), r.Payload["password"]); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	case UPDATE_SQUAD_NAME:
		if _, ok := r.Payload["squadId"]; !ok {
			http.Error(w, "no field squadId in payload", http.StatusBadRequest)
			return
		}
		if _, ok := r.Payload["squadName"]; !ok {
			http.Error(w, "no field squadName in payload", http.StatusBadRequest)
			return
		}
		if err = m.UpdateSquadName(r.Payload["squadId"],r.Payload["squadName"]); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	case UPDATE_SQUAD_PASSWORD:
		if _, ok := r.Payload["squadId"]; !ok {
			http.Error(w, "no field squadId in payload", http.StatusBadRequest)
			return
		}
		if _, ok := r.Payload["password"]; !ok {
			http.Error(w, "no field password in payload", http.StatusBadRequest)
			return
		}
		if err = m.UpdateSquadPassword(r.Payload["squadId"],r.Payload["password"]); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	case UPDATE_SQUAD_AUTHORIZED_MEMBERS:
		if _, ok := r.Payload["squadId"]; !ok {
			http.Error(w, "no field squadId in payload", http.StatusBadRequest)
			return
		}
		if _, ok := r.Payload["authorizedMember"]; !ok {
			http.Error(w, "no field authorizedMember in payload", http.StatusBadRequest)
			return
		}
		if err = m.UpdateSquadAuthorizedMembers(r.Payload["squadId"],r.Payload["authorizedMember"]); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	}
	return
}
