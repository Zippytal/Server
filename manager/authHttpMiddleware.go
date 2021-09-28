package manager

import (
	"context"
	"net/http"
)

const (
	AUTHENTICATE      = "authenticate"
	PEER_AUTH_INIT    = "peer_auth_init"
	PEER_AUTH_VERIFY  = "peer_auth_verify"
	NODE_AUTH_INIT    = "node_auth_init"
	NNODE_AUTH_VERIFY = "node_auth_verify"
)

type AuthHTTPMiddleware struct {
	manager *AuthManager
}

func (am *AuthHTTPMiddleware) Process(ctx context.Context, r *ServRequest, req *http.Request, w http.ResponseWriter) (err error) {
	switch r.Type {
	}
	return
}
