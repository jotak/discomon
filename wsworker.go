package main

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type wsworker struct {
	eventChan chan Event
}

const (
	// Time allowed to write the file to the client.
	writeWait = 10 * time.Second
	// Time allowed to read the next pong message from the client.
	pongWait = 60 * time.Second
	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

var (
	master = Master()
	wsWorkers = syncWorkers{}
)

func Master() (*wsworker) {
	master := wsworker{make(chan Event, 10)}
	go func() {
    for {
			event := <- master.eventChan
			wsWorkers.ForEach(func(w *wsworker) { w.eventChan <- event })
		}
	} ()
	return &master
}

func startWorker(ws *websocket.Conn) {
	w := &wsworker{make(chan Event, 10)}
	go w.writer(ws)
	wsWorkers.Push(w)
	w.reader(ws)
}

func (w *wsworker) reader(ws *websocket.Conn) {
	defer ws.Close()
	ws.SetReadLimit(512)
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (w *wsworker) writer(ws *websocket.Conn) {
	pingTicker := time.NewTicker(pingPeriod)
	defer func() {
		pingTicker.Stop()
		ws.Close()
	}()
	log.Println("[init] Sending data over WS")
	sendDashboardsLinks(ws)
	sendInventory(ws)
	for {
		select {
		case event := <-w.eventChan:
			var err error
			if event.Type == EventDashChanged {
				log.Println("Sending dashboards over WS")
				err = sendDashboardsLinks(ws)
			} else if event.Type == EventInvChanged {
				log.Println("Sending inventory over WS")
				err = sendInventory(ws)
			} else {
				ws.SetWriteDeadline(time.Now().Add(writeWait))
				var obj interface{}
				if event.Type == EventLog {
					obj = struct {
						Log string
					}{event.Attachment.(string),}
				} else {
					obj = struct {
						LastScan time.Time
					}{event.Attachment.(time.Time),}
				}
				err = ws.WriteJSON(obj)
			}
			if err != nil {
				log.Printf("WS failure: %v\n", err)
				return
			}
		case <-pingTicker.C:
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				log.Printf("WS failure: %v\n", err)
				return
			}
		}
	}
}

type Link struct {
	Link string
	Name string
}

func sendDashboardsLinks(ws *websocket.Conn) error {
	links := []Link{}
	for dash, _ := range dashboards {
		l := Link{grafanaExternalUrl + "/dashboard/db/" + dash, dash}
		links = append(links, l)
	}
	ws.SetWriteDeadline(time.Now().Add(writeWait))
	var obj = struct {
		Dashboards []Link
	}{ links }
	return ws.WriteJSON(obj)
}

func sendInventory(ws *websocket.Conn) error {
	ws.SetWriteDeadline(time.Now().Add(writeWait))
	var obj = struct {
		Inventory Resource
	}{ inventory }
	return ws.WriteJSON(obj)
}

type syncWorkers struct {
	sync.Mutex
	workers []*wsworker
}
func (workers *syncWorkers) Push(w *wsworker) {
	workers.Lock()
	defer workers.Unlock()

  workers.workers = append(workers.workers, w)
}
func (workers *syncWorkers) ForEach(f func(*wsworker)) {
	workers.Lock()
	defer workers.Unlock()
	for _, w := range workers.workers {
		f(w)
	}
}
