package main

import (
	"github.com/skyhookml/skyhookml/skyhook"

	_ "github.com/skyhookml/skyhookml/ops"

	"fmt"
	"log"
	"net"
	"net/http"
	"os"
)

func main() {
	var coordinatorURL string
	var execOp skyhook.ExecOp

	var bindAddr string = ":8080"
	if len(os.Args) >= 2 {
		bindAddr = os.Args[1]
	}

	http.HandleFunc("/exec/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(404)
			return
		}

		var request skyhook.ExecBeginRequest
		if err := skyhook.ParseJsonRequest(w, r, &request); err != nil {
			http.Error(w, fmt.Sprintf("error parsing request: %v", err), 400)
			return
		}

		coordinatorURL = request.CoordinatorURL
		var err error
		execOp, err = request.Node.GetOp().Prepare(coordinatorURL, request.Node)
		if err != nil {
			http.Error(w, fmt.Sprintf("error preparing node: %v", err), 400)
			return
		}

		skyhook.JsonResponse(w, skyhook.ExecBeginResponse{
			Parallelism: execOp.Parallelism(),
		})
	})

	http.HandleFunc("/exec/task", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(404)
			return
		}

		var request skyhook.ExecTaskRequest
		if err := skyhook.ParseJsonRequest(w, r, &request); err != nil {
			http.Error(w, fmt.Sprintf("error parsing request: %v", err), 400)
			return
		}

		err := execOp.Apply(request.Task)
		if err != nil {
			log.Printf("[container] apply error: %v", err)
			http.Error(w, err.Error(), 400)
			return
		}
	})

	http.HandleFunc("/exit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(404)
			return
		}
		os.Exit(0)
	})

	ln, err := net.Listen("tcp", bindAddr)
	if err != nil {
		panic(err)
	}
	fmt.Println("ready")
	if err := http.Serve(ln, nil); err != nil {
		panic(err)
	}
}
