package manager

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type HostedSquadDBManager struct {
	*mongo.Collection
}

const HOSTED_SQUAD_COLLECTION_NAME = "hosted_squads"

func NewHostedSquadDBManager(host string, port int) (hostedDBManager *HostedSquadDBManager, err error) {
	hostedSquadDBManagerCh, errCh := make(chan *HostedSquadDBManager), make(chan error)
	go func() {
		dbManagerCh, errC := NewDbManager(context.Background(), DB_NAME, host, port)
		select {
		case dbManager := <-dbManagerCh:
			hostedSquadDBManagerCh <- &HostedSquadDBManager{dbManager.Db.Collection(HOSTED_SQUAD_COLLECTION_NAME)}
		case e := <-errC:
			errCh <- e
		}
	}()
	select {
	case err = <-errCh:
		return
	case hostedDBManager = <-hostedSquadDBManagerCh:
		return
	}
}

func (pdm *HostedSquadDBManager) AddNewHostedSquad(ctx context.Context, squad *Squad) (err error) {
	var p Squad
	if err = pdm.FindOne(ctx, bson.M{"id": squad.ID}).Decode(&p); err == nil {
		err = fmt.Errorf("A hosted squad with id %s already exist", squad.ID)
		return
	}
	_, err = pdm.InsertOne(ctx, squad)
	return
}

func (pdm *HostedSquadDBManager) GetHostedSquad(ctx context.Context, peerId string) (squad *Squad, err error) {
	var s Squad
	err = pdm.FindOne(ctx, bson.M{"id": peerId}).Decode(&s)
	squad = &s
	return
}

func (pdm *HostedSquadDBManager) GetHostedSquads(ctx context.Context, limit int64, lastIndex int64) (squads []*Squad, err error) {
	res, err := pdm.Find(ctx, bson.D{}, options.Find().SetLimit(limit).SetSkip(lastIndex))
	if err != nil {
		return
	}
	err = res.All(ctx, &squads)
	return
}

func (pdm *HostedSquadDBManager) GetHostedSquadsByName(ctx context.Context, pattern string, limit int64, lastIndex int64) (squads []*Squad, err error) {
	res, err := pdm.Find(ctx, bson.D{{"name", primitive.Regex{Pattern: pattern, Options: ""}}}, options.Find().SetLimit(limit).SetSkip(lastIndex))
	if err != nil {
		return
	}
	err = res.All(ctx, &squads)
	return
}

func (pdm *HostedSquadDBManager) GetHostedSquadsByID(ctx context.Context, pattern string, limit int64, lastIndex int64) (squads []*Squad, err error) {
	res, err := pdm.Find(ctx, bson.D{{"id", primitive.Regex{Pattern: pattern, Options: ""}}}, options.Find().SetLimit(limit).SetSkip(lastIndex))
	if err != nil {
		return
	}
	err = res.All(ctx, &squads)
	return
}

func (pdm *HostedSquadDBManager) GetHostedSquadsByOwner(ctx context.Context, owner string, limit int64, lastIndex int64) (squads []*Squad, err error) {
	res, err := pdm.Find(ctx, bson.M{"owner": owner}, options.Find().SetLimit(limit).SetSkip(lastIndex))
	if err != nil {
		return
	}
	err = res.All(ctx, &squads)
	return
}

func (pdm *HostedSquadDBManager) GetHostedSquadsByHost(ctx context.Context, host string, limit int64, lastIndex int64) (squads []*Squad, err error) {
	res, err := pdm.Find(ctx, bson.M{"hostid": host}, options.Find().SetLimit(limit).SetSkip(lastIndex))
	if err != nil {
		return
	}
	err = res.All(ctx, &squads)
	return
}

func (pdm *HostedSquadDBManager) DeleteHostedSquad(ctx context.Context, squadId string) (err error) {
	_, err = pdm.DeleteOne(ctx, bson.M{"id": squadId})
	return
}

func (pdm *HostedSquadDBManager) UpdateHostedSquadName(ctx context.Context, squadId string, newName string) (err error) {
	_, err = pdm.UpdateOne(ctx, bson.M{"id": squadId}, bson.D{
		{"$set", bson.D{{"name", newName}}},
	})
	return
}

func (pdm *HostedSquadDBManager) UpdateHostedSquadStatus(ctx context.Context, squadId string, newStatus bool) (err error) {
	_, err = pdm.UpdateOne(ctx, bson.M{"id": squadId}, bson.D{
		{"$set", bson.D{{"status", newStatus}}},
	})
	return
}

func (pdm *HostedSquadDBManager) UpdateHostedSquadMembers(ctx context.Context, squadId string, members []string) (err error) {
	_, err = pdm.UpdateOne(ctx, bson.M{"id": squadId}, bson.D{
		{"$set", bson.D{{"members", members}}},
	})
	return
}

func (pdm *HostedSquadDBManager) UpdateHostedSquadAuthorizedMembers(ctx context.Context, squadId string, authorizedMembers []string) (err error) {
	_, err = pdm.UpdateOne(ctx, bson.M{"id": squadId}, bson.D{
		{"$set", bson.D{{"authorizedMembers", authorizedMembers}}},
	})
	return
}
