package woolsocks

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	readWait   = 60 * time.Second
	pingPeriod = (readWait * 9) / 10
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Msg struct {
	Type    string      `json:"type"`
	Message interface{} `json:"msg"`
}

type Conn struct {
	ws  *websocket.Conn
	ID  string
	In  chan *Msg
	Out chan *Msg
}

func (c *Conn) reader() {
	defer func() {
		c.ws.Close()
	}()
	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			break
		}
		var m Msg
		if err := json.Unmarshal(message, &m); err != nil {
			continue
		}
		c.In <- &m
	}
}

func (c *Conn) writer() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()
	for {
		select {
		case message, ok := <-c.Out:
			if !ok {
				c.write(websocket.CloseMessage, []byte{})
				return
			}
			messageBytes, err := json.Marshal(message)
			if err != nil {
				log.Println("Marshal error: " + err.Error())
				continue
			}
			if err := c.write(websocket.TextMessage, messageBytes); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func (c *Conn) write(mt int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}

type WoolSocks struct {
	Conns chan *Conn
}

func New() *WoolSocks {
	ws := &WoolSocks{
		Conns: make(chan *Conn, 0),
	}
	return ws
}

func (wool *WoolSocks) Handler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	c := &Conn{
		ws:  ws,
		ID:  r.URL.Query().Get("id"),
		In:  make(chan *Msg, 0),
		Out: make(chan *Msg, 0),
	}

	c.ws.SetPongHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(writeWait))
		return nil
	})
	wool.Conns <- c
	go c.writer()
	c.reader()
	close(c.In)
	close(c.Out)
}
