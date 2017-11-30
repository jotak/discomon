package main

import (
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type Link struct {
	Link string
	Name string
}

var (
	// Time allowed to write the file to the client.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the client.
	pongWait = 60 * time.Second

	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	upgrader  = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	logChan = make(chan string)
	lastScanChan = make(chan time.Time)
	dashChan = make(chan int)
	invChan = make(chan int)
)

func initServer() {
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", serveWs)
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))
	log.Println("Server listening on port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", 404)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var v = struct {
		Host    string
		Data    string
	}{
		r.Host,
		"",
	}
	t, _ := template.ParseFiles("assets/view.html")
	t.Execute(w, &v)
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return
	}

	go writer(ws)
	reader(ws)
}

func reader(ws *websocket.Conn) {
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

func writer(ws *websocket.Conn) {
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
		case msg := <-logChan:
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			var obj = struct {
				Log string
			}{msg,}
			if err := ws.WriteJSON(obj); err != nil {
				return
			}
		case timestamp := <-lastScanChan:
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			var obj = struct {
				LastScan time.Time
			}{timestamp,}
			if err := ws.WriteJSON(obj); err != nil {
				return
			}
		case <-dashChan:
			log.Println("Sending dashboards over WS")
			if err := sendDashboardsLinks(ws); err != nil {
				return
			}
		case <-invChan:
			log.Println("Sending inventory over WS")
			if err := sendInventory(ws); err != nil {
				return
			}
		case <-pingTicker.C:
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
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

func logch(msg string) {
	select {
	case logChan <- msg:
	default:
	}
}

func dashch() {
	select {
	case dashChan <- 1:
	default:
	}
}

func invch() {
	select {
	case invChan <- 1:
	default:
	}
}

func scanch() {
	select {
	case lastScanChan <- time.Now():
	default:
	}
}
