package manager

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ZoneDBManager struct {
	*mongo.Collection
}

const ZONE_COLLECTION_NAME = "zones"

func NewZoneDBManager(host string, port int) (hostedDBManager *ZoneDBManager, err error) {
	ZoneDBManagerCh, errCh := make(chan *ZoneDBManager), make(chan error)
	go func() {
		dbManagerCh, errC := NewDbManager(context.Background(), DB_NAME, host, port)
		select {
		case dbManager := <-dbManagerCh:
			ZoneDBManagerCh <- &ZoneDBManager{dbManager.Db.Collection(HOSTED_SQUAD_COLLECTION_NAME)}
		case e := <-errC:
			errCh <- e
		}
	}()
	select {
	case err = <-errCh:
		return
	case hostedDBManager = <-ZoneDBManagerCh:
		return
	}
}

func (pdm *ZoneDBManager) AddNewZone(ctx context.Context, zone *Zone) (err error) {
	var p Zone
	if err = pdm.FindOne(ctx, bson.M{"id": zone.ID}).Decode(&p); err == nil {
		err = fmt.Errorf("A hosted zone with id %s already exist", zone.ID)
		return
	}
	_, err = pdm.InsertOne(ctx, zone)
	return
}

func (pdm *ZoneDBManager) GetZone(ctx context.Context, zoneId string) (zone *Zone, err error) {
	var s Zone
	err = pdm.FindOne(ctx, bson.M{"id": zoneId}).Decode(&s)
	zone = &s
	return
}

func (pdm *ZoneDBManager) GetZones(ctx context.Context, limit int64, lastIndex int64) (zones []*Zone, err error) {
	res, err := pdm.Find(ctx, bson.D{}, options.Find().SetLimit(limit).SetSkip(lastIndex))
	if err != nil {
		return
	}
	err = res.All(ctx, &zones)
	return
}

func (pdm *ZoneDBManager) GetZonesByName(ctx context.Context, pattern string, limit int64, lastIndex int64) (zones []*Zone, err error) {
	res, err := pdm.Find(ctx, bson.D{{"name", primitive.Regex{Pattern: pattern, Options: ""}}}, options.Find().SetLimit(limit).SetSkip(lastIndex))
	if err != nil {
		return
	}
	err = res.All(ctx, &zones)
	return
}

func (pdm *ZoneDBManager) GetZonesByID(ctx context.Context, pattern string, limit int64, lastIndex int64) (zones []*Zone, err error) {
	res, err := pdm.Find(ctx, bson.D{{"id", primitive.Regex{Pattern: pattern, Options: ""}}}, options.Find().SetLimit(limit).SetSkip(lastIndex))
	if err != nil {
		return
	}
	err = res.All(ctx, &zones)
	return
}

func (pdm *ZoneDBManager) GetZoneByID(ctx context.Context, id string) (zone *Zone, err error) {
	err = pdm.FindOne(ctx, bson.M{"id": id}).Decode(&zone)
	return
}

func (pdm *ZoneDBManager) GetZonesByOwner(ctx context.Context, owner string, limit int64, lastIndex int64) (zones []*Zone, err error) {
	res, err := pdm.Find(ctx, bson.M{"owner": owner}, options.Find().SetLimit(limit).SetSkip(lastIndex))
	if err != nil {
		return
	}
	err = res.All(ctx, &zones)
	return
}

func (pdm *ZoneDBManager) GetZonesByHost(ctx context.Context, host string, limit int64, lastIndex int64) (zones []*Zone, err error) {
	res, err := pdm.Find(ctx, bson.M{"hostid": host}, options.Find().SetLimit(limit).SetSkip(lastIndex))
	if err != nil {
		return
	}
	err = res.All(ctx, &zones)
	return
}

func (pdm *ZoneDBManager) DeleteZone(ctx context.Context, zoneId string) (err error) {
	_, err = pdm.DeleteOne(ctx, bson.M{"id": zoneId})
	return
}

func (pdm *ZoneDBManager) UpdateZoneName(ctx context.Context, zoneId string, newName string) (err error) {
	_, err = pdm.UpdateOne(ctx, bson.M{"id": zoneId}, bson.D{
		{"$set", bson.D{{"name", newName}}},
	})
	return
}

func (pdm *ZoneDBManager) UpdateZonePassword(ctx context.Context, zoneId string, newPassword string) (err error) {
	_, err = pdm.UpdateOne(ctx, bson.M{"id": zoneId}, bson.D{
		{"$set", bson.D{{"password", newPassword}}},
	})
	return
}

func (pdm *ZoneDBManager) UpdateZoneStatus(ctx context.Context, zoneId string, newStatus bool) (err error) {
	_, err = pdm.UpdateOne(ctx, bson.M{"id": zoneId}, bson.D{
		{"$set", bson.D{{"status", newStatus}}},
	})
	return
}

func (pdm *ZoneDBManager) UpdateZoneMembers(ctx context.Context, zoneId string, members []string) (err error) {
	_, err = pdm.UpdateOne(ctx, bson.M{"id": zoneId}, bson.D{
		{"$set", bson.D{{"members", members}}},
	})
	return
}

func (pdm *ZoneDBManager) UpdateZoneAuthorizedMembers(ctx context.Context, zoneId string, authorizedMembers []string) (err error) {
	_, err = pdm.UpdateOne(ctx, bson.M{"id": zoneId}, bson.D{
		{"$set", bson.D{{"authorizedMembers", authorizedMembers}}},
	})
	return
}
