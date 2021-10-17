package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type ServRequest struct {
	Type    string            `json:"type"`
	To      string            `json:"to"`
	From    string            `json:"from"`
	Token   string            `json:"token"`
	Payload map[string]string `json:"payload"`
}

type WSMiddleware interface {
	Process(*ServRequest, *Manager, *websocket.Conn) error
}

type HTTPMiddleware interface {
	Process(context.Context, *ServRequest, *http.Request, http.ResponseWriter) error
}

type WSHandler struct {
	wsMiddlewares   []WSMiddleware
	httpMiddlewares []HTTPMiddleware
	manager         *Manager
	path            string
}

type WSServ struct {
	Server *http.Server
}

func NewWSServ(addr string, handler http.Handler) (wsServ *WSServ) {
	s := &http.Server{
		Addr:    addr,
		Handler: handler,
	}
	wsServ = &WSServ{
		Server: s,
	}
	return
}

func NewWSHandler(path string, manager *Manager, wsMiddlewares []WSMiddleware, httpMiddlewares []HTTPMiddleware) (wsHandler *WSHandler) {
	wsHandler = &WSHandler{
		wsMiddlewares:   wsMiddlewares,
		httpMiddlewares: httpMiddlewares,
		manager:         manager,
		path:            path,
	}
	return
}

func (wsh *WSHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	done, errCh := make(chan struct{}), make(chan error)
	var peerId string
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("recover from panic in serve http : %v\n", r)
		}
	}()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("recover from panic in serve http : %v\n", r)
			}
		}()
		switch req.URL.Path {
		case "/ws":
			conn, err := upgrader.Upgrade(w, req, nil)
			if err != nil {
				errCh <- err
				return
			}
			defer conn.Close()
			doneCh, msgCh := make(chan struct{}), make(chan []byte)
			conn.SetCloseHandler(func(code int, text string) error {
				close(doneCh)
				close(msgCh)
				wsh.manager.Lock()
				if _, ok := wsh.manager.WSPeers[peerId]; ok {
					wsh.manager.WSPeers[peerId].mux.Lock()
					if wsh.manager.WSPeers[peerId].CurrentZoneId != "" {
						_ = wsh.manager.LeaveZone(wsh.manager.AuthManager.AuthTokenValid[peerId], wsh.manager.WSPeers[peerId].CurrentZoneId, peerId)
					}
					if wsh.manager.WSPeers[peerId].CurrentSquadId != "" {
						_ = wsh.manager.LeaveSquad(wsh.manager.WSPeers[peerId].CurrentSquadId, peerId, MESH)
					}
					if wsh.manager.WSPeers[peerId].CurrentHostedSquadId != "" {
						_ = wsh.manager.LeaveSquad(wsh.manager.WSPeers[peerId].CurrentHostedSquadId, peerId, HOSTED)
					}
					if wsh.manager.WSPeers[peerId].CurrentCallId != "" {
						if _, ok := wsh.manager.WSPeers[wsh.manager.WSPeers[peerId].CurrentCallId]; ok {
							_ = wsh.manager.WSPeers[wsh.manager.WSPeers[peerId].CurrentCallId].Conn.WriteJSON(map[string]interface{}{
								"type": "stop_call",
								"from": peerId,
								"payload": map[string]string{
									"userId": peerId,
								},
							})
							wsh.manager.WSPeers[wsh.manager.WSPeers[peerId].CurrentCallId].CurrentCallId = ""
							_ = wsh.manager.RemoveIncomingCall(peerId, wsh.manager.WSPeers[peerId].CurrentCallId)
						}
					}
					_ = wsh.manager.RemoveIncomingCall(wsh.manager.WSPeers[peerId].CurrentCallId, peerId)
					wsh.manager.WSPeers[peerId].mux.Unlock()
				}
				delete(wsh.manager.WSPeers, peerId)
				wsh.manager.Unlock()
				return nil
			})
			go func() {
				for msg := range msgCh {
					fmt.Printf("sending msg %v to dst\n", msg)
					var req ServRequest
					if err := json.Unmarshal(msg, &req); err != nil {
						log.Println(err)
						return
					}
					if req.Type == WS_INIT {
						peerId = req.From
					}
					fmt.Println("my cool request", req)
					for _, middleware := range wsh.wsMiddlewares {
						if err := middleware.Process(&req, wsh.manager, conn); err != nil {
							log.Println(err)
							continue
						}
					}
				}
			}()
			for {
				_, message, err := conn.ReadMessage()
				if err != nil {
					errCh <- err
					conn.Close()
					break
				}
				fmt.Printf("received message %s\n", string(message))
				select {
				case msgCh <- message:
				case <-done:
					return
				}
			}
		case "/req":
			fmt.Println("got req", req.Body)
			body, err := io.ReadAll(req.Body)
			if err != nil {
				errCh <- err
				return
			}
			var r ServRequest
			if err := json.Unmarshal(body, &r); err != nil {
				log.Println(err)
				return
			}
			wg := &sync.WaitGroup{}
			for _, httpMiddleware := range wsh.httpMiddlewares {
				wg.Add(1)
				go func(hm HTTPMiddleware) {
					if err := hm.Process(req.Context(), &r, req, w); err != nil {
						log.Println(err)
					}
					wg.Done()
				}(httpMiddleware)
			}
			wg.Wait()
		default:
			if _, err := os.Stat(fmt.Sprintf("./%s/%s", wsh.path, req.URL.Path)); os.IsNotExist(err) {
				http.ServeFile(w, req, fmt.Sprintf("./%s/index.html", wsh.path))
			} else {
				http.ServeFile(w, req, fmt.Sprintf("./%s/%s", wsh.path, req.URL.Path))
			}
		}
		done <- struct{}{}
	}()
	select {
	case <-req.Context().Done():
		log.Println("context error :", req.Context().Err())
		return
	case <-done:
		return
	case err := <-errCh:
		log.Println(err)
		return
	}
}
