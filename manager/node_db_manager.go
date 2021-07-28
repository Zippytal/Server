package manager

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type NodeDBManager struct {
	*mongo.Collection
}

const NODE_COLLECTION_NAME = "nodes"

func NewNodeDBManager(host string, port int) (nodeDBManager *NodeDBManager, err error) {
	nodeDBManagerCh, errCh := make(chan *NodeDBManager), make(chan error)
	go func() {
		dbManagerCh, errC := NewDbManager(context.Background(), DB_NAME, host, port)
		select {
		case dbManager := <-dbManagerCh:
			nodeDBManagerCh <- &NodeDBManager{dbManager.Db.Collection(NODE_COLLECTION_NAME)}
		case e := <-errC:
			errCh <- e
		}
	}()
	select {
	case err = <-errCh:
		return
	case nodeDBManager = <-nodeDBManagerCh:
		return
	}
}

func (pdm *NodeDBManager) AddNewNode(ctx context.Context, node *Node) (err error) {
	var p Node
	if err = pdm.FindOne(ctx, bson.M{"id": node.Id}).Decode(&p); err == nil {
		err = fmt.Errorf("A node with id %s already exist", node.Id)
		return
	}
	_, err = pdm.InsertOne(ctx, node)
	return
}

func (pdm *NodeDBManager) GetNode(ctx context.Context, nodeId string) (node *Node, err error) {
	err = pdm.FindOne(ctx, bson.M{"id": nodeId}).Decode(&node)
	return
}

func (pdm *NodeDBManager) GetNodes(ctx context.Context, limit int64, lastIndex int64) (nodes []*Node, err error) {
	res, err := pdm.Find(ctx, bson.D{}, options.Find().SetLimit(limit).SetSkip(lastIndex))
	if err != nil {
		return
	}
	err = res.All(ctx, &nodes)
	return
}

func (pdm *NodeDBManager) GetNodesByName(ctx context.Context, pattern string, limit int64, lastIndex int64) (nodes []*Node, err error) {
	res, err := pdm.Find(ctx, bson.D{{"name", primitive.Regex{Pattern: pattern, Options: ""}}}, options.Find().SetLimit(limit).SetSkip(lastIndex))
	if err != nil {
		return
	}
	err = res.All(ctx, &nodes)
	return
}

func (pdm *NodeDBManager) GetNodesByID(ctx context.Context, pattern string, limit int64, lastIndex int64) (nodes []*Node, err error) {
	res, err := pdm.Find(ctx, bson.D{{"id", primitive.Regex{Pattern: pattern, Options: ""}}}, options.Find().SetLimit(limit).SetSkip(lastIndex))
	if err != nil {
		return
	}
	err = res.All(ctx, &nodes)
	return
}

func (pdm *NodeDBManager) DeleteNode(ctx context.Context, nodeId string) (err error) {
	_, err = pdm.DeleteOne(ctx, bson.M{"id": nodeId})
	return
}

func (pdm *NodeDBManager) UpdateNodeName(ctx context.Context, nodeId string, newName string) (err error) {
	_, err = pdm.UpdateOne(ctx, bson.M{"id": nodeId}, bson.D{
		{"$set", bson.D{{"name", newName}}},
	})
	return
}

func (pdm *NodeDBManager) UpdateNodeStatus(ctx context.Context, nodeId string, newStatus bool) (err error) {
	_, err = pdm.UpdateOne(ctx, bson.M{"id": nodeId}, bson.D{
		{"$set", bson.D{{"status", newStatus}}},
	})
	return
}

func (pdm *NodeDBManager) UpdateNodeFriends(ctx context.Context, nodeId string, friends []string) (err error) {

	_, err = pdm.UpdateOne(ctx, bson.M{"id": nodeId}, bson.D{
		{"$set", bson.D{{"friends", friends}}},
	})
	return
}

func (pdm *NodeDBManager) UpdateNodeFriendRequests(ctx context.Context, nodeId string, requests []string) (err error) {
	fmt.Println(requests)
	_, err = pdm.UpdateOne(ctx, bson.M{"id": nodeId}, bson.D{
		{"$set", bson.D{{"friendRequests", requests}}},
	})
	return
}
