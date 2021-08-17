package manager

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SquadDBManager struct {
	*mongo.Collection
}

const SQUAD_COLLECTION_NAME = "squads"

func NewSquadDBManager(host string, port int) (squadDBManager *SquadDBManager, err error) {
	squadDBManagerCh, errCh := make(chan *SquadDBManager), make(chan error)
	go func() {
		dbManagerCh, errC := NewDbManager(context.Background(), DB_NAME, host, port)
		select {
		case dbManager := <-dbManagerCh:
			squadDBManagerCh <- &SquadDBManager{dbManager.Db.Collection(SQUAD_COLLECTION_NAME)}
		case e := <-errC:
			errCh <- e
		}
	}()
	select {
	case err = <-errCh:
		return
	case squadDBManager = <-squadDBManagerCh:
		return
	}
}

func (pdm *SquadDBManager) AddNewSquad(ctx context.Context, squad *Squad) (err error) {
	var p Squad
	if err = pdm.FindOne(ctx, bson.M{"id": squad.ID}).Decode(&p); err == nil {
		err = fmt.Errorf("A squad with id %s already exist", squad.ID)
		return
	}
	_, err = pdm.InsertOne(ctx, squad)
	return
}

func (pdm *SquadDBManager) GetSquad(ctx context.Context, peerId string) (squad *Squad, err error) {
	var s Squad
	err = pdm.FindOne(ctx, bson.M{"id": peerId}).Decode(&s)
	squad = &s
	return
}

func (pdm *SquadDBManager) GetSquads(ctx context.Context, limit int64, lastIndex int64) (squads []*Squad, err error) {
	res, err := pdm.Find(ctx, bson.M{}, options.Find().SetLimit(limit).SetSkip(lastIndex))
	if err != nil {
		return
	}
	err = res.All(ctx, &squads)
	return
}

func (pdm *SquadDBManager) GetSquadsByName(ctx context.Context, pattern string, limit int64, lastIndex int64) (squads []*Squad, err error) {
	res, err := pdm.Find(ctx, bson.D{{"name", primitive.Regex{Pattern: pattern, Options: ""}}}, options.Find().SetLimit(limit).SetSkip(lastIndex))
	if err != nil {
		return
	}
	err = res.All(ctx, &squads)
	return
}

func (pdm *SquadDBManager) GetSquadByID(ctx context.Context, id string) (squad *Squad, err error) {
	err = pdm.FindOne(ctx, bson.M{"id": id}).Decode(&squad)
	return
}

func (pdm *SquadDBManager) GetSquadsByID(ctx context.Context, pattern string, limit int64, lastIndex int64) (squads []*Squad, err error) {
	res, err := pdm.Find(ctx, bson.D{{"id", primitive.Regex{Pattern: pattern, Options: ""}}}, options.Find().SetLimit(limit).SetSkip(lastIndex))
	if err != nil {
		return
	}
	err = res.All(ctx, &squads)
	return
}

func (pdm *SquadDBManager) GetSquadsByOwner(ctx context.Context, owner string, limit int64, lastIndex int64) (squads []*Squad, err error) {
	res, err := pdm.Find(ctx, bson.M{"owner": owner}, options.Find().SetLimit(limit).SetSkip(lastIndex))
	if err != nil {
		return
	}
	err = res.All(ctx, &squads)
	return
}

func (pdm *SquadDBManager) DeleteSquad(ctx context.Context, squadId string) (err error) {
	_, err = pdm.DeleteOne(ctx, bson.M{"id": squadId})
	return
}

func (pdm *SquadDBManager) UpdateSquadName(ctx context.Context, squadId string, newName string) (err error) {
	_, err = pdm.UpdateOne(ctx, bson.M{"id": squadId}, bson.D{
		{"$set", bson.D{{"name", newName}}},
	})
	return
}

func (pdm *SquadDBManager) UpdateSquadStatus(ctx context.Context, squadId string, newStatus bool) (err error) {
	_, err = pdm.UpdateOne(ctx, bson.M{"id": squadId}, bson.D{
		{"$set", bson.D{{"status", newStatus}}},
	})
	return
}

func (pdm *SquadDBManager) UpdateSquadMembers(ctx context.Context, squadId string, members []string) (err error) {
	_, err = pdm.UpdateOne(ctx, bson.M{"id": squadId}, bson.D{
		{"$set", bson.D{{"members", members}}},
	})
	return
}

func (pdm *SquadDBManager) UpdateSquadAuthorizedMembers(ctx context.Context, squadId string, authorizedMembers []string) (err error) {
	_, err = pdm.UpdateOne(ctx, bson.M{"id": squadId}, bson.D{
		{"$set", bson.D{{"authorizedMembers", authorizedMembers}}},
	})
	return
}
