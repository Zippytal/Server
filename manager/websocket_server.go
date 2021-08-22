package manager

import (
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
	Process(*ServRequest, *http.Request, http.ResponseWriter, *Manager) error
}

type WSHandler struct {
	wsMiddlewares   []WSMiddleware
	httpMiddlewares []HTTPMiddleware
	manager         *Manager
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

func NewWSHandler(manager *Manager, wsMiddlewares []WSMiddleware, httpMiddlewares []HTTPMiddleware) (wsHandler *WSHandler) {
	wsHandler = &WSHandler{
		wsMiddlewares:   wsMiddlewares,
		httpMiddlewares: httpMiddlewares,
		manager:         manager,
	}
	return
}

func (wsh *WSHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	done, errCh := make(chan struct{}), make(chan error)
	var peerId string
	go func() {
		switch req.URL.Path {
		case "/ws":
			conn, err := upgrader.Upgrade(w, req, nil)
			if err != nil {
				errCh <- err
				return
			}
			defer conn.Close()
			doneCh, msgCh := make(chan struct{}), make(chan []byte, 100)
			defer close(doneCh)
			conn.SetCloseHandler(func(code int, text string) error {
				if _, ok := wsh.manager.WSPeers[peerId]; ok {
					wsh.manager.WSPeers[peerId].mux.Lock()
					_ = wsh.manager.LeaveSquad(wsh.manager.WSPeers[peerId].CurrentSquadId, peerId, MESH)
					_ = wsh.manager.LeaveSquad(wsh.manager.WSPeers[peerId].CurrentHostedSquadId, peerId, HOSTED)
					if _, ok := wsh.manager.WSPeers[wsh.manager.WSPeers[peerId].CurrentCallId]; ok {
						_ = wsh.manager.WSPeers[wsh.manager.WSPeers[peerId].CurrentCallId].Conn.WriteJSON(map[string]interface{}{
							"type": "stop_call",
							"from": peerId,
							"payload": map[string]string{
								"userId": peerId,
							},
						})
					}
					_ = wsh.manager.RemoveIncomingCall(wsh.manager.WSPeers[peerId].CurrentCallId, peerId)
					wsh.manager.WSPeers[peerId].mux.Unlock()
				}
				delete(wsh.manager.WSPeers, peerId)
				close(doneCh)
				close(msgCh)
				return nil
			})
			go func() {
				for msg := range msgCh {
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
							return
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
					if err := hm.Process(&r, req, w, wsh.manager); err != nil {
						log.Println(err)
					}
					wg.Done()
				}(httpMiddleware)
			}
			wg.Wait()
		default:
			if _, err := os.Stat("./app/" + req.URL.Path); os.IsNotExist(err) {
				http.ServeFile(w, req, "./app/index.html")
			} else {
				http.ServeFile(w, req, "./app/"+req.URL.Path)
			}
		}
		done <- struct{}{}
	}()
	select {
	case <-req.Context().Done():
		log.Println(req.Context().Err())
		return
	case <-done:
		return
	case err := <-errCh:
		log.Println(err)
		return
	}
}
