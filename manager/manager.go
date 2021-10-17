package manager

import (
	"fmt"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type (
	ManagerState  uint8
	GRPCPeerState uint8
	WSState       uint8
	SquadType     string
	SquadEvent    string

	GRPCPeer struct {
		Conn        GrpcManager_LinkServer
		State       GRPCPeerState
		DisplayName string
		DbPeer      *Peer
		mux         *sync.Mutex
	}

	WSPeer struct {
		Conn                 *websocket.Conn
		DisplayName          string
		DbPeer               *Peer
		State                WSState
		CurrentSquadId       string
		CurrentHostedSquadId string
		CurrentCallId        string
		CurrentZoneId        string
		mux                  *sync.Mutex
	}

	Manager struct {
		State       ManagerState
		GRPCPeers   map[string]*GRPCPeer
		WSPeers     map[string]*WSPeer
		Squads      map[string]*Squad
		AuthManager *AuthManager
		*NodeManager
		*ZoneManager
		*PeerManager
		*SquadManager
		*HostedSquadManager
		*sync.RWMutex
	}
)

const (
	PRIVATE SquadType = "private"
	PUBLIC  SquadType = "public"
)

const (
	INCOMING_MEMBER                 SquadEvent = "incoming_member"
	HOSTED_INCOMING_MEMBER          SquadEvent = "hosted_incoming_member"
	LEAVING_MEMBER                  SquadEvent = "leaving_member"
	HOSTED_LEAVING_MEMBER           SquadEvent = "hosted_leaving_member"
	NEW_HOSTED_SQUAD                           = "new_hosted_squad"
	NEW_SQUAD                                  = "new_squad"
	NEW_FRIEND_REQUEST                         = "new_friend_request"
	DELETE_FRIEND                              = "delete_friend"
	NEW_AUTHORIZED_HOSTED_SQUAD                = "new_authorized_hosted_squad"
	NEW_AUTHORIZED_SQUAD                       = "new_authorized_squad"
	REMOVED_AUTHORIZED_HOSTED_SQUAD            = "removed_authorized_hosted_squad"
	REMOVED_AUTHORIZED_SQUAD                   = "removed_authorized_squad"
)

const (
	ON ManagerState = iota
	OFF
)

const (
	CONNECTED GRPCPeerState = iota
	SLEEP
)

const DB_NAME string = "zippytal_server"

func NewManager() (manager *Manager, err error) {
	authManager := NewAuthManager()
	manager = &Manager{
		State:       ON,
		GRPCPeers:   make(map[string]*GRPCPeer),
		WSPeers:     make(map[string]*WSPeer),
		Squads:      make(map[string]*Squad),
		RWMutex:     &sync.RWMutex{},
		AuthManager: authManager,
	}
	squadManager, err := NewSquadManager(manager, authManager)
	if err != nil {
		log.Println(err)
		return
	}
	hostedSquadManager, err := NewHostedSquadManager(manager, authManager)
	if err != nil {
		log.Println(err)
		return
	}
	peerManager, err := NewPeerManager(manager, authManager)
	if err != nil {
		log.Println(err)
		return
	}
	nodeManager, err := NewNodeManager(manager, authManager)
	if err != nil {
		log.Println(err)
		return
	}
	zoneManager, err := NewZoneManager(manager, authManager)
	if err != nil {
		log.Println(err)
		return
	}
	manager.ZoneManager = zoneManager
	manager.SquadManager = squadManager
	manager.HostedSquadManager = hostedSquadManager
	manager.PeerManager = peerManager
	manager.NodeManager = nodeManager
	return
}

func (manager *Manager) AddGrpcPeer(peer GrpcManager_LinkServer, id string, req *Request) (err error) {
	fmt.Printf("adding peer %s\n", req.From)
	manager.Lock()
	manager.GRPCPeers[req.From] = &GRPCPeer{Conn: peer, State: CONNECTED, mux: new(sync.Mutex)}
	manager.Unlock()
	if _, ok := req.Payload["to"]; ok {
		if _, ok := manager.GRPCPeers[req.From]; ok {
			if err = manager.GRPCPeers[req.From].Conn.Send(&Response{
				Type:    req.Type,
				Success: true,
				Payload: req.Payload,
			}); err != nil {
				return
			}
		}
	}
	err = manager.manage(peer)
	delete(manager.GRPCPeers, req.From)
	return
}

func (manager *Manager) manage(peer GrpcManager_LinkServer) (err error) {
	done, errch := make(chan struct{}), make(chan error)
	go func() {
		for {
			req, err := peer.Recv()
			if err != nil {
				errch <- err
				return
			}
			fmt.Println(req)
			if _, ok := req.Payload["to"]; ok {
				to := req.Payload["to"]
				if _, ok := manager.GRPCPeers[to]; ok {
					manager.GRPCPeers[to].mux.Lock()
					if err := manager.GRPCPeers[to].Conn.Send(&Response{
						Type:    req.Type,
						Success: true,
						Payload: req.Payload,
					}); err != nil {
						manager.GRPCPeers[to].mux.Unlock()
						errch <- err
						return
					}
					manager.GRPCPeers[to].mux.Unlock()
				} else if _, ok := manager.WSPeers[to]; ok {
					manager.WSPeers[to].mux.Lock()
					if err = manager.WSPeers[to].Conn.WriteJSON(map[string]interface{}{
						"from":    req.From,
						"to":      to,
						"type":    req.Type,
						"payload": req.Payload,
					}); err != nil {
						log.Println(err)
						manager.WSPeers[to].mux.Unlock()
						return
					}
					manager.WSPeers[to].mux.Unlock()
				}
			}
		}
	}()
	select {
	case <-done:
		log.Println("manage is done")
		return
	case err = <-errch:
		return
	case <-peer.Context().Done():
		err = peer.Context().Err()
		return
	}
}
