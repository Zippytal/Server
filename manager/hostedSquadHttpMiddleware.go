package manager

import (
	"encoding/json"
	"net/http"
	"strconv"
)

const (
	JOIN_HOSTED_SQUAD                      = "join_hosted_squad"
	LIST_HOSTED_SQUADS                     = "list_hosted_squads"
	LIST_HOSTED_SQUADS_BY_NAME             = "list_hosted_squads_by_name"
	LIST_HOSTED_SQUADS_BY_ID               = "list_hosted_squads_by_id"
	GET_HOSTED_SQUADS_BY_OWNER             = "get_hosted_squads_by_owner"
	LIST_HOSTED_SQUADS_BY_HOST             = "list_hosted_squads_by_host"
	HOSTED_SQUAD_ACCESS_DENIED             = "squad_access_denied"
	HOSTED_SQUAD_ACCESS_GRANTED            = "squad_access_granted"
	LEAVE_HOSTED_SQUAD                     = "leave_hosted_squad"
	HOSTED_SQUAD_AUTH                      = "auth_hosted_squad"
	CREATE_HOSTED_SQUAD                    = "create_hosted_squad"
	DELETE_HOSTED_SQUAD                    = "delete_hosted_squad"
	MODIFY_HOSTED_SQUAD                    = "modify_hosted_squad"
	UPDATE_HOSTED_SQUAD_NAME               = "update_hosted_squad_name"
	UPDATE_HOSTED_SQUAD_AUTHORIZED_MEMBERS = "update_hosted_squad_authorized_members"
	DELETE_HOSTED_SQUAD_AUTHORIZED_MEMBERS = "delete_hosted_squad_authorized_members"
	UPDATE_HOSTED_SQUAD_PASSWORD           = "update_hosted_squad_password"
)

type HostedSquadHTTPMiddleware struct{}

func (shm *HostedSquadHTTPMiddleware) Process(r *ServRequest, req *http.Request, w http.ResponseWriter, m *Manager) (err error) {
	switch r.Type {
	case GET_HOSTED_SQUADS_BY_OWNER:
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
	case JOIN_HOSTED_SQUAD:
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
	case LIST_HOSTED_SQUADS:
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
	case LIST_HOSTED_SQUADS_BY_NAME:
		if err = VerifyFields(r.Payload, "squadName", "networkType", "lastIndex"); err != nil {
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
	case LIST_HOSTED_SQUADS_BY_HOST:
		if err = VerifyFields(r.Payload, "host", "lastIndex"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		lastIndex, err := strconv.Atoi(r.Payload["lastIndex"])
		if err != nil {
			http.Error(w, "provide a valid integer for last index", http.StatusBadRequest)
			return err
		}
		squads, err := m.ListSquadsByHost(int64(lastIndex), r.Payload["host"], HOSTED)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}
		err = json.NewEncoder(w).Encode(squads)
	case LIST_HOSTED_SQUADS_BY_ID:
		if err = VerifyFields(r.Payload, "squadId", "networkType", "lastIndex"); err != nil {
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
	case LEAVE_HOSTED_SQUAD:
		if err = VerifyFields(r.Payload, "squadId", "squadNetworkType"); err != nil {
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
	case CREATE_HOSTED_SQUAD:
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
	case DELETE_HOSTED_SQUAD:
		if err = VerifyFields(r.Payload, "squadId"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err = m.DeleteSquad(r.Token, r.Payload["squadId"], r.From, HOSTED); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	case MODIFY_HOSTED_SQUAD:
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
	case UPDATE_HOSTED_SQUAD_NAME:
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
	case UPDATE_HOSTED_SQUAD_PASSWORD:
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
	case UPDATE_HOSTED_SQUAD_AUTHORIZED_MEMBERS:
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
	case DELETE_HOSTED_SQUAD_AUTHORIZED_MEMBERS:
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
