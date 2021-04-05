package main

import (
	"github.com/skyhookml/skyhookml/app"
	"github.com/skyhookml/skyhookml/skyhook"

	_ "github.com/skyhookml/skyhookml/ops"

	"github.com/googollee/go-socket.io"

	"flag"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
)

func main() {
	addr := flag.String("addr", ":8080", "bind address")
	coordinatorURL := flag.String("url", "http://127.0.0.1:PORT", "coordinator URL, workers must be able to reach it")
	initdb := flag.Bool("initdb", false, "initialize the database before starting up")
	workerURL := flag.String("worker", "http://127.0.0.1:8081", "worker or worker-pool URL")
	instanceID := flag.String("instance-id", "", "instance ID")
	flag.Parse()

	tcpAddr, err := net.ResolveTCPAddr("tcp", *addr)
	if err != nil {
		panic(err)
	}

	app.Config.CoordinatorURL = strings.ReplaceAll(*coordinatorURL, "PORT", strconv.Itoa(tcpAddr.Port))
	app.Config.WorkerURL = *workerURL
	app.Config.InstanceID = *instanceID

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	skyhook.SeedRand()

	app.InitDB(*initdb)

	server, err := socketio.NewServer(nil)
	if err != nil {
		panic(err)
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
	log.Printf("starting on %s", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		panic(err)
	}
}
