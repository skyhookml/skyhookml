package main

import (
	"./app"
	"./skyhook"

	_ "./train_ops/keras"
	_ "./exec_ops/model"
	_ "./exec_ops/python"
	_ "./exec_ops/render"
	_ "./exec_ops/video_sample"

	"github.com/googollee/go-socket.io"

	"log"
	"net/http"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	skyhook.SeedRand()
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
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
