package manager

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PeerDBManager struct {
	*mongo.Collection
}

const PEER_COLLECTION_NAME = "peers"

func NewPeerDBManager(host string, port int) (peerDBManager *PeerDBManager, err error) {
	peerDBManagerCh, errCh := make(chan *PeerDBManager), make(chan error)
	go func() {
		dbManagerCh, errC := NewDbManager(context.Background(), DB_NAME, host, port)
		select {
		case dbManager := <-dbManagerCh:
			peerDBManagerCh <- &PeerDBManager{dbManager.Db.Collection(PEER_COLLECTION_NAME)}
		case e := <-errC:
			errCh <- e
		}
	}()
	select {
	case err = <-errCh:
		return
	case peerDBManager = <-peerDBManagerCh:
		return
	}
}

func (pdm *PeerDBManager) AddNewPeer(ctx context.Context, peer *Peer) (err error) {
	var p Peer
	if err = pdm.FindOne(ctx, bson.M{"id": peer.Id}).Decode(&p); err == nil {
		err = fmt.Errorf("A peer with id %s already exist", peer.Id)
		return
	}
	_, err = pdm.InsertOne(ctx, peer)
	return
}

func (pdm *PeerDBManager) GetPeer(ctx context.Context, peerId string) (peer *Peer, err error) {
	err = pdm.FindOne(ctx, bson.M{"id": peerId}).Decode(&peer)
	return
}

func (pdm *PeerDBManager) GetPeers(ctx context.Context, limit int64, lastIndex int64) (peers []*Peer, err error) {
	res, err := pdm.Find(ctx, bson.D{}, options.Find().SetLimit(limit).SetSkip(lastIndex))
	if err != nil {
		return
	}
	err = res.All(ctx, &peers)
	return
}

func (pdm *PeerDBManager) GetPeersByName(ctx context.Context, pattern string, limit int64, lastIndex int64) (peers []*Peer, err error) {
	res, err := pdm.Find(ctx, bson.D{{"name", primitive.Regex{Pattern: pattern, Options: ""}}}, options.Find().SetLimit(limit).SetSkip(lastIndex))
	if err != nil {
		return
	}
	err = res.All(ctx, &peers)
	return
}

func (pdm *PeerDBManager) GetPeersByID(ctx context.Context, pattern string, limit int64, lastIndex int64) (peers []*Peer, err error) {
	res, err := pdm.Find(ctx, bson.D{{"id", primitive.Regex{Pattern: pattern, Options: ""}}}, options.Find().SetLimit(limit).SetSkip(lastIndex))
	if err != nil {
		return
	}
	err = res.All(ctx, &peers)
	return
}

func (pdm *PeerDBManager) DeletePeer(ctx context.Context, peerId string) (err error) {
	_, err = pdm.DeleteOne(ctx, bson.M{"id": peerId})
	return
}

func (pdm *PeerDBManager) UpdatePeerName(ctx context.Context, peerId string, newName string) (err error) {
	_, err = pdm.UpdateOne(ctx, bson.M{"id": peerId}, bson.D{
		{"$set", bson.D{{"name", newName}}},
	})
	return
}

func (pdm *PeerDBManager) UpdatePeerStatus(ctx context.Context, peerId string, newStatus bool) (err error) {
	_, err = pdm.UpdateOne(ctx, bson.M{"id": peerId}, bson.D{
		{"$set", bson.D{{"status", newStatus}}},
	})
	return
}

func (pdm *PeerDBManager) UpdatePeerFriends(ctx context.Context, peerId string, friends []string) (err error) {
	_, err = pdm.UpdateOne(ctx, bson.M{"id": peerId}, bson.D{
		{"$set", bson.D{{"friends", friends}}},
	})
	return
}

func (pdm *PeerDBManager) UpdateKnownSquads(ctx context.Context, peerId string, knownSquads []string) (err error) {
	_, err = pdm.UpdateOne(ctx, bson.M{"id": peerId}, bson.D{
		{"$set", bson.D{{"knownsquadsid", knownSquads}}},
	})
	return
}

func (pdm *PeerDBManager) UpdateKnownHostedSquads(ctx context.Context, peerId string, knownHostedSquads []string) (err error) {
	_, err = pdm.UpdateOne(ctx, bson.M{"id": peerId}, bson.D{
		{"$set", bson.D{{"knownhostedsquadsid", knownHostedSquads}}},
	})
	return
}
func (pdm *PeerDBManager) UpdateKnownZones(ctx context.Context, peerId string, knownZonesId []string) (err error) {
	_, err = pdm.UpdateOne(ctx, bson.M{"id": peerId}, bson.D{
		{"$set", bson.D{{"knownzonesid", knownZonesId}}},
	})
	return
}

func (pdm *PeerDBManager) UpdatePeerFriendRequests(ctx context.Context, peerId string, requests []string) (err error) {
	_, err = pdm.UpdateOne(ctx, bson.M{"id": peerId}, bson.D{
		{"$set", bson.D{{"friendrequests", requests}}},
	})
	return
}

func (pdm *PeerDBManager) UpdateIncomingCalls(ctx context.Context, peerId string, incomingCalls []string) (err error) {
	_, err = pdm.UpdateOne(ctx, bson.M{"id": peerId}, bson.D{
		{"$set", bson.D{{"incomingcalls", incomingCalls}}},
	})
	return
}
