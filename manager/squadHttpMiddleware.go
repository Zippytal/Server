package manager

import (
	"encoding/json"
	"net/http"
	"strconv"
)

const (
	JOIN_SQUAD                      = "join_squad"
	LIST_SQUADS                     = "list_squads"
	LIST_SQUADS_BY_NAME             = "list_squads_by_name"
	LIST_SQUADS_BY_ID               = "list_squads_by_id"
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
	DELETE_SQUAD_AUTHORIZED_MEMBERS = "delete_squad_authorized_members"
	UPDATE_SQUAD_PASSWORD           = "update_squad_password"
)

type SquadHTTPMiddleware struct{}

func (shm *SquadHTTPMiddleware) Process(r *ServRequest, req *http.Request, w http.ResponseWriter, m *Manager) (err error) {
	switch r.Type {
	case GET_SQUADS_BY_OWNER:
		if err = VerifyFields(r.Payload, "owner", "lastIndex", "networkType"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		lastIndex, err := strconv.Atoi(r.Payload["lastIndex"])
		if err != nil {
			http.Error(w, "field lastIndex is not an int", http.StatusBadRequest)
			return err
		}
		squads, err := m.GetSquadSByOwner(r.Token, r.Payload["owner"], int64(lastIndex), r.Payload["networkType"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return err
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"squads":  squads,
		})
	case JOIN_SQUAD:
		if err = VerifyFields(r.Payload, "squadId", "password", "networkType"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
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
		if err = VerifyFields(r.Payload, "networkType", "lastIndex"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
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
		if err = VerifyFields(r.Payload, "networkType", "lastIndex", "squadName"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
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
		if err = VerifyFields(r.Payload, "networkType", "lastIndex", "squadId"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
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
		if err = VerifyFields(r.Payload, "squadNetworkType", "squadId"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
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
		if err = VerifyFields(r.Payload, "squadId", "password", "squadType", "squadName", "squadNetworkType"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
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
		if err = VerifyFields(r.Payload, "squadId"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err = m.DeleteSquad(r.Token, r.Payload["squadId"], r.From, MESH); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	case MODIFY_SQUAD:
		if err = VerifyFields(r.Payload, "squadId", "password", "squadName", "squadType"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
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
		if err = VerifyFields(r.Payload, "squadId", "squadName", "networkType"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err = m.UpdateSquadName(r.Payload["squadId"], r.Payload["squadName"], r.Payload["networkType"]); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	case UPDATE_SQUAD_PASSWORD:
		if err = VerifyFields(r.Payload, "squadId", "password"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err = m.UpdateSquadPassword(r.Payload["squadId"], r.Payload["password"]); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	case UPDATE_SQUAD_AUTHORIZED_MEMBERS:
		if err = VerifyFields(r.Payload, "squadId", "authorizedMember", "networkType"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err = m.UpdateSquadAuthorizedMembers(r.Payload["squadId"], r.Payload["authorizedMember"], r.Payload["networkType"]); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	case DELETE_SQUAD_AUTHORIZED_MEMBERS:
		if err = VerifyFields(r.Payload, "squadId", "authorizedMember", "networkType"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err = m.DeleteSquadAuthorizedMembers(r.Payload["squadId"], r.Payload["authorizedMember"], r.Payload["networkType"]); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	}
	return
}
