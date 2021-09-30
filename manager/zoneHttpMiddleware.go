package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

const (
	NEW_AUTHORIZED_ZONE            = "new_authorized_zone"
	REMOVED_AUTHORIZED_ZONE        = "removed_authorized_zone"
	INCOMING_ZONE_MEMBER           = "incoming_zone_member"
	ZONE_MEMBER_DISCONNECTED       = "zone_member_disconnected"
	NEW_ZONE                       = "new_zone"
	JOIN_ZONE                      = "join_zone"
	LIST_ZONES                     = "list_zones"
	LIST_ZONES_BY_NAME             = "list_zones_by_name"
	LIST_ZONES_BY_ID               = "list_zones_by_id"
	GET_ZONE_BY_ID                 = "get_zone_by_id"
	GET_ZONES_BY_OWNER             = "get_zones_by_owner"
	LIST_ZONES_BY_HOST             = "list_zones_by_host"
	ZONE_ACCESS_DENIED             = "squad_access_denied"
	ZONE_ACCESS_GRANTED            = "squad_access_granted"
	LEAVE_ZONE                     = "leave_zone"
	ZONE_AUTH                      = "auth_zone"
	CREATE_ZONE                    = "create_zone"
	DELETE_ZONE                    = "delete_zone"
	MODIFY_ZONE                    = "modify_zone"
	UPDATE_ZONE_NAME               = "update_zone_name"
	UPDATE_ZONE_AUTHORIZED_MEMBERS = "update_zone_authorized_members"
	DELETE_ZONE_AUTHORIZED_MEMBERS = "delete_zone_authorized_members"
	UPDATE_ZONE_PASSWORD           = "update_zone_password"
)

type ZoneHTTPMiddleware struct {
	manager *Manager
}

func NewZoneHTTPMiddleware(manager *Manager) *ZoneHTTPMiddleware {
	return &ZoneHTTPMiddleware{
		manager: manager,
	}
}

func (zhm *ZoneHTTPMiddleware) Process(ctx context.Context, r *ServRequest, req *http.Request, w http.ResponseWriter) (err error) {
	fmt.Println("zone http middleware called")
	switch r.Type {
	case GET_ZONES_BY_OWNER:
		if err = VerifyFields(r.Payload, "owner", "lastIndex"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		lastIndex, err := strconv.Atoi(r.Payload["lastIndex"])
		if err != nil {
			http.Error(w, "field lastIndex is not an int", http.StatusBadRequest)
			return err
		}
		zones, err := zhm.manager.GetZoneByOwner(r.Token, r.Payload["owner"], int64(lastIndex))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return err
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"zones":   zones,
		})
	case JOIN_ZONE:
		if err = VerifyFields(r.Payload, "zoneId", "password"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err = zhm.manager.ConnectToZone(r.Token, r.Payload["zoneId"], r.From, r.Payload["password"]); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"zoneId":  r.Payload["zoneId"],
		})
	case LIST_ZONES:
		if err = VerifyFields(r.Payload, "lastIndex"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		lastIndex, err := strconv.Atoi(r.Payload["lastIndex"])
		if err != nil {
			http.Error(w, "provide a valid integer for last index", http.StatusBadRequest)
			return err
		}
		zones, err := zhm.manager.ListAllZones(int64(lastIndex))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}
		err = json.NewEncoder(w).Encode(zones)
	case LIST_ZONES_BY_NAME:
		if err = VerifyFields(r.Payload, "zoneName", "lastIndex"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		lastIndex, err := strconv.Atoi(r.Payload["lastIndex"])
		if err != nil {
			http.Error(w, "provide a valid integer for last index", http.StatusBadRequest)
			return err
		}
		zones, err := zhm.manager.ListZonesByName(int64(lastIndex), r.Payload["zoneName"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}
		err = json.NewEncoder(w).Encode(zones)
	case LIST_ZONES_BY_HOST:
		if err = VerifyFields(r.Payload, "host", "lastIndex"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		lastIndex, err := strconv.Atoi(r.Payload["lastIndex"])
		if err != nil {
			http.Error(w, "provide a valid integer for last index", http.StatusBadRequest)
			return err
		}
		zones, err := zhm.manager.ListZonesByHost(int64(lastIndex), r.Payload["host"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}
		err = json.NewEncoder(w).Encode(zones)
	case LIST_ZONES_BY_ID:
		if err = VerifyFields(r.Payload, "zoneId", "lastIndex"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		lastIndex, err := strconv.Atoi(r.Payload["lastIndex"])
		if err != nil {
			http.Error(w, "provide a valid integer for last index", http.StatusBadRequest)
			return err
		}
		zones, err := zhm.manager.ListZonesByID(int64(lastIndex), r.Payload["zoneId"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}
		err = json.NewEncoder(w).Encode(zones)
	case GET_ZONE_BY_ID:
		if err = VerifyFields(r.Payload, "zoneId"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		squad, err := zhm.manager.GetZoneByID(r.Payload["zoneId"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}
		err = json.NewEncoder(w).Encode(squad)
	case LEAVE_ZONE:
		if err = VerifyFields(r.Payload, "zoneId"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err = zhm.manager.LeaveZone(r.Token,r.Payload["zoneId"], r.From); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"zoneId":  r.Payload["zoneId"],
		})
	case SQUAD_AUTH:
	case CREATE_ZONE:
		if err = VerifyFields(r.Payload, "zoneId", "password", "zoneName","zoneHost"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if _, ok := r.Payload["zoneHost"]; !ok {
			http.Error(w, "no field zoneHost in payload", http.StatusBadRequest)
			return
		}
		if err = zhm.manager.CreateZone(r.Token, r.Payload["zoneId"], r.From, r.Payload["zoneName"], r.Payload["password"], r.Payload["zoneHost"]); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	case DELETE_ZONE:
		if err = VerifyFields(r.Payload, "zoneId"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err = zhm.manager.DeleteZone(r.Token, r.Payload["zoneId"], r.From); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	case MODIFY_ZONE:
		if err = VerifyFields(r.Payload, "zoneId", "password", "zoneName", "squadType"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err = zhm.manager.ModifyZone(r.Token, r.Payload["zoneId"], r.From, r.Payload["zoneName"], r.Payload["password"]); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	case UPDATE_ZONE_NAME:
		if err = VerifyFields(r.Payload, "zoneId", "zoneName"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err = zhm.manager.UpdateZoneName(r.Payload["zoneId"], r.Payload["zoneName"]); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	case UPDATE_ZONE_PASSWORD:
		if err = VerifyFields(r.Payload, "zoneId", "password"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err = zhm.manager.UpdateZonePassword(r.Payload["zoneId"], r.Payload["password"]); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	case UPDATE_ZONE_AUTHORIZED_MEMBERS:
		if err = VerifyFields(r.Payload, "zoneId", "authorizedMember"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err = zhm.manager.UpdateZoneAuthorizedMembers(r.Payload["zoneId"], r.Payload["authorizedMember"]); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	case DELETE_ZONE_AUTHORIZED_MEMBERS:
		if err = VerifyFields(r.Payload, "zoneId", "authorizedMember"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err = zhm.manager.DeleteZoneAuthorizedMembers(r.Payload["zoneId"], r.Payload["authorizedMember"]); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	}
	return
}
