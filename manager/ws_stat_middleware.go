package manager

import (
	"fmt"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

const (
	WS_INIT string  = "init"
	WS_OPEN WSState = 1
)

type WSStateMiddleware struct {
	lock *sync.Mutex
}

func NewWSStateMiddleware() *WSStateMiddleware {
	return &WSStateMiddleware{
		lock: &sync.Mutex{},
	}
}

func (wsm *WSStateMiddleware) Process(req *ServRequest, manager *Manager, conn *websocket.Conn) (err error) {
	switch req.Type {
	case WS_INIT:
		manager.Lock()
		if id, ok := manager.AuthManager.AuthTokenValid[req.Token]; ok {
			if id == req.From {
				if p, err := manager.GetPeer(req.From); err == nil {
					manager.WSPeers[req.From] = &WSPeer{
						State:                WS_OPEN,
						DbPeer:               p,
						Conn:                 conn,
						DisplayName:          p.Name,
						mux:                  &sync.Mutex{},
						CurrentSquadId:       "",
						CurrentHostedSquadId: "",
						CurrentCallId:        "",
					}
					manager.Unlock()
					return err
				}
			}
		}
		manager.WSPeers[req.From] = &WSPeer{
			State:                WS_OPEN,
			Conn:                 conn,
			mux:                  &sync.Mutex{},
			DisplayName:          "anonymous",
			CurrentSquadId:       "",
			CurrentHostedSquadId: "",
			CurrentCallId:        "",
		}
		manager.Unlock()
		return
	default:
		fmt.Println(manager.WSPeers)
		if ws, ok := manager.WSPeers[req.To]; ok {
			wsm.lock.Lock()
			if err = ws.Conn.WriteJSON(map[string]interface{}{
				"from":    req.From,
				"to":      req.To,
				"type":    req.Type,
				"payload": req.Payload,
			}); err != nil {
				log.Println(err)
				wsm.lock.Unlock()
				return
			}
			wsm.lock.Unlock()
		} else if grpc, ok := manager.GRPCPeers[req.To]; ok {
			fmt.Println("seding to grpc peer")
			payload := make(map[string]string)
			for i, v := range req.Payload {
				payload[i] = v
			}
			payload["to"] = req.To
			payload["from"] = req.From
			manager.GRPCPeers[req.To].mux.Lock()
			if err = grpc.Conn.Send(&Response{
				Type:    req.Type,
				Success: true,
				Payload: payload,
			}); err != nil {
				log.Println(err)
				manager.GRPCPeers[req.To].mux.Unlock()
				return
			}
			manager.GRPCPeers[req.To].mux.Unlock()
		} else {
			err = fmt.Errorf("no corresponding peer for id %s", req.To)
			return
		}
	}
	return
}
