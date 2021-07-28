package manager

import (
	"encoding/json"
	"net/http"
)

const (
	CREATE_NODE = "create_node"
	GET_NODE    = "get_node"
	DELETE_NODE = "delete_node"
	SET_NODE_ACTIVE
)

type NodeHTTPMiddleware struct{}

func (shm *NodeHTTPMiddleware) Process(r *ServRequest, req *http.Request, w http.ResponseWriter, m *Manager) (err error) {
	switch r.Type {
	case CREATE_NODE:
		if err = VerifyFields(r.Payload, "token", "peerId"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err = m.Authenticate(r.Payload["peerId"], r.Payload["token"]); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]string{
			"success": "true",
		})
	case GET_NODE:
		if err = VerifyFields(r.Payload, "peerId"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		token, tokenErr := m.PeerAuthInit(r.Payload["peerId"])
		if tokenErr != nil {
			http.Error(w, tokenErr.Error(), http.StatusInternalServerError)
			return tokenErr
		}
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"peerId":  r.Payload["peerId"],
			"token":   token,
		})
	case DELETE_NODE:
		if err = VerifyFields(r.Payload, "token", "peerId"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
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
	}
	return
}
