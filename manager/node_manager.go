package manager

import "context"

type NodeManager struct {
	DB          *NodeDBManager
	manager     *Manager
	authManager *AuthManager
}

func NewNodeManager(manager *Manager, authManager *AuthManager) (nodeManager *NodeManager, err error) {
	nodeDBManager, err := NewNodeDBManager("localhost", 27017)
	if err != nil {
		return
	}
	nodeManager = &NodeManager{
		DB:          nodeDBManager,
		manager:     manager,
		authManager: authManager,
	}
	return
}

func (nm *NodeManager) CreateNode(nodeId string, nodeKey string, nodeUsername string) (err error) {
	node := &Node{
		PubKey:         nodeKey,
		Id:             nodeId,
		Name:           nodeUsername,
		Friends:        []string{},
		KnownPeersId:   []string{},
		FriendRequests: []string{},
		Active:         true,
	}
	err = nm.DB.AddNewNode(context.Background(), node)
	return
}
