package manager

import (
	"encoding/json"
	"net/http"
)

const (
	AUTHENTICATE      = "authenticate"
	PEER_AUTH_INIT    = "peer_auth_init"
	PEER_AUTH_VERIFY  = "peer_auth_verify"
	NODE_AUTH_INIT    = "node_auth_init"
	NNODE_AUTH_VERIFY = "node_auth_verify"
)

type AuthHTTPMiddleware struct{}

func (shm *AuthHTTPMiddleware) Process(r *ServRequest, req *http.Request, w http.ResponseWriter, m *Manager) (err error) {
	switch r.Type {
	case AUTHENTICATE:
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
	case PEER_AUTH_INIT:
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
	case PEER_AUTH_VERIFY:
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
