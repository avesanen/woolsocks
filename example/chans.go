package main

import (
	"log"
	"net/http"

	"github.com/avesanen/woolsocks"
)

func ConnHandler(c *woolsocks.Conn) {
	for m := range c.In {
		log.Println(m.Message)
		c.Out <- m
	}
	log.Println("Conn exited.")
}

func main() {
	ws := woolsocks.New()
	http.HandleFunc("/", ws.Handler)
	go http.ListenAndServe("0.0.0.0:8080", nil)

	for {
		c, ok := <-ws.Conns
		if !ok {
			return
		}
		go ConnHandler(c)
	}
}
