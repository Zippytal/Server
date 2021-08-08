package manager

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
)

const (
	INIT = "init"
	TEST = "test"
)

type GRPCManagerService struct {
	*UnimplementedGrpcManagerServer
	Manager *Manager
}

func NewGRPCManagerService(manager *Manager) (service *GRPCManagerService) {
	service = &GRPCManagerService{
		Manager: manager,
	}
	return
}

func (service *GRPCManagerService) Link(stream GrpcManager_LinkServer) (err error) {
	done, errch := make(chan struct{}), make(chan error)
	fmt.Println("link init")
	go func() {
		req, err := stream.Recv()
		if err != nil {
			errch <- err
			return
		} else if service.Manager.State == ON {
			if err := service.Manager.AddGrpcPeer(stream, req.From, req); err != nil {
				errch <- err
				return
			}
			done <- struct{}{}
		}
	}()
	select {
	case <-stream.Context().Done():
		err = stream.Context().Err()
		log.Println(err)
		return
	case <-done:
		log.Println("Link with peer is successful")
		return
	case err = <-errch:
		log.Println(err)
		return
	}
}

func (service *GRPCManagerService) ListPeers(ctx context.Context, peerListRequest *PeerListRequest) (peerListResponse *PeerListResponse, err error) {
	list, errch := make(chan []*Peer), make(chan error)
	go func() {
		peerList := []*Peer{}
		count := 0
		for id := range service.Manager.GRPCPeers {
			if count < int(peerListRequest.Number) {
				peer := &Peer{
					Id:   id,
					Name: "unknown",
				}
				peerList = append(peerList, peer)
			} else {
				break
			}
		}
		for id := range service.Manager.WSPeers {
			if count < int(peerListRequest.Number) {
				peer := &Peer{
					Id:   id,
					Name: "unknown",
				}
				peerList = append(peerList, peer)
			} else {
				break
			}
		}
		list <- peerList
	}()
	select {
	case <-ctx.Done():
		log.Println(err)
		err = ctx.Err()
		return
	case peerList := <-list:
		peerListResponse = &PeerListResponse{
			Peers:     peerList,
			Success:   true,
			LastIndex: peerListRequest.LastIndex + int32(len(peerList)),
		}
		return
	case err = <-errch:
		log.Println(err)
		return
	}
}

func (service *GRPCManagerService) CreateSquad(ctx context.Context, req *SquadCreateRequest) (res *SquadCreateResponse, err error) {
	done, errch := make(chan *SquadCreateResponse), make(chan error)
	go func() {
		uid, uidErr := uuid.NewUUID()
		if uidErr != nil {
			errch <- uidErr
			return
		}
		if err := service.Manager.CreateSquad(req.Token, uid.String(), req.UserId, req.Name, SquadType(req.SquadType), req.Password, "mesh", "lolo_local_serv"); err != nil {
			errch <- err
			return
		}
		done <- &SquadCreateResponse{
			Success: true,
			Reason:  "Squad creation succes",
			Squad: &ProtoSquad{
				Name:    req.Name,
				Id:      uid.String(),
				Members: make([]string, 0),
				Owner:   req.UserId,
			},
		}
	}()
	select {
	case <-ctx.Done():
		err = ctx.Err()
		return
	case err = <-errch:
		return
	case res = <-done:
		return
	}
}

func (service *GRPCManagerService) UpdateSquad(ctx context.Context, req *SquadUpdateRequest) (res *SquadUpdateResponse, err error) {
	done, errch := make(chan *SquadUpdateResponse), make(chan error)
	go func() {
		if err := service.Manager.ModifySquad(req.Token, req.Id, req.UserId, req.Name, SquadType(req.SquadType), req.Password); err != nil {
			errch <- err
			return
		}
		done <- &SquadUpdateResponse{
			Success: true,
			Reason:  fmt.Sprintf("Squad %s updated", req.Id),
			Squad: &ProtoSquad{
				Id:        req.Id,
				Name:      req.Name,
				Members:   service.Manager.Squads[req.Id].Members,
				Owner:     req.UserId,
				SquadType: req.SquadType,
			},
		}
	}()
	select {
	case <-ctx.Done():
		err = ctx.Err()
		return
	case err = <-errch:
		return
	case res = <-done:
		return
	}
}

func (service *GRPCManagerService) DeleteSquad(ctx context.Context, req *SquadDeleteRequest) (res *SquadDeleteResponse, err error) {
	done, errch := make(chan *SquadDeleteResponse), make(chan error)
	go func() {
		if err := service.Manager.DeleteSquad(req.Token, req.SquadId, req.UserId, MESH); err != nil {
			errch <- err
			return
		}
		done <- &SquadDeleteResponse{
			Succes: true,
			Reason: fmt.Sprintf("Squad %s deleted", req.SquadId),
			Squad:  &ProtoSquad{},
		}
	}()
	select {
	case <-ctx.Done():
		err = ctx.Err()
		return
	case err = <-errch:
		return
	case res = <-done:
		return
	}
}

func (service *GRPCManagerService) ListSquad(ctx context.Context, req *SquadListRequest) (res *SquadListResponse, err error) {
	done, errch := make(chan *SquadListResponse), make(chan error)
	go func() {
		squadList := make([]*ProtoSquad, 0, req.Number)
		count := 0
		for _, squad := range service.Manager.Squads {
			if count < int(req.Number) {
				squadList = append(squadList, &ProtoSquad{
					Id:        squad.ID,
					Name:      squad.Name,
					Owner:     squad.Owner,
					Members:   squad.Members,
					SquadType: string(squad.SquadType),
				})
			} else {
				break
			}
		}
		done <- &SquadListResponse{
			Success:   true,
			LastIndex: req.Number,
			Squads:    squadList,
		}
	}()
	select {
	case <-ctx.Done():
		err = ctx.Err()
		return
	case err = <-errch:
		return
	case res = <-done:
		return
	}
}

func (service *GRPCManagerService) ConnectSquad(ctx context.Context, req *SquadConnectRequest) (res *SquadConnectResponse, err error) {
	done, errch := make(chan *SquadConnectResponse), make(chan error)
	go func() {
		if err := service.Manager.ConnectToSquad(req.Token, req.Id, req.UserId, req.Password, ""); err != nil {
			errch <- err
			return
		}
		done <- &SquadConnectResponse{
			Success: true,
			Reason:  fmt.Sprintf("connected to squad %s", req.Id),
			Id:      req.Id,
		}
	}()
	select {
	case <-ctx.Done():
		err = ctx.Err()
		return
	case err = <-errch:
		return
	case res = <-done:
		return
	}
}

func (service *GRPCManagerService) LeaveSquad(ctx context.Context, req *SquadLeaveRequest) (res *SquadLeaveResponse, err error) {
	done, errch := make(chan *SquadLeaveResponse), make(chan error)
	go func() {
		if err := service.Manager.LeaveSquad(req.SquadId, req.UserId, ""); err != nil {
			errch <- err
			return
		}
		done <- &SquadLeaveResponse{
			Success: true,
			Reason:  fmt.Sprintf("left squad %s", req.SquadId),
			SquadId: req.SquadId,
		}
	}()
	select {
	case <-ctx.Done():
		err = ctx.Err()
		return
	case err = <-errch:
		return
	case res = <-done:
		return
	}
}
