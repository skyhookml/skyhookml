package main

import (
	"./app"
	"./skyhook"

	_ "./ops"

	"github.com/googollee/go-socket.io"

	"log"
	"net/http"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	skyhook.SeedRand()
	server := socketio.NewServer(nil)
	server.OnConnect("/", func(s socketio.Conn) error {
		return nil
	})
	for _, f := range app.SetupFuncs {
		f(server)
	}

	go server.Serve()
	defer server.Close()
	http.Handle("/socket.io/", server)
	http.Handle("/", app.Router)
	log.Printf("starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
