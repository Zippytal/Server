package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	CREATE_NODE = "create_node"
	GET_NODE    = "get_node"
	DELETE_NODE = "delete_node"
	SET_NODE_ACTIVE
)

type NodeHTTPMiddleware struct {
	manager *Manager
}

func NewNodeHTTPMiddleware(manager *Manager) *NodeHTTPMiddleware {
	return &NodeHTTPMiddleware{
		manager: manager,
	}
}

func (shm *NodeHTTPMiddleware) Process(ctx context.Context, r *ServRequest, req *http.Request, w http.ResponseWriter) (err error) {
	fmt.Println("node middleware called")
	fmt.Println(r.From, r.Type)
	switch r.Type {
	case CREATE_NODE:
		fmt.Println("create node called")
		if err = VerifyFields(r.Payload, "nodeId", "nodeKey", "nodeUsername"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err = shm.manager.CreateNode(r.Payload["nodeId"], r.Payload["nodeKey"], r.Payload["nodeUsername"]); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = json.NewEncoder(w).Encode(map[string]string{
			"success": "true",
		})
	case GET_NODE:
	case DELETE_NODE:
	}
	return
}
