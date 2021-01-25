package main

import (
	"./skyhook"

	_ "./ops"

	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
)

func main() {
	var coordinatorURL string
	var execOp skyhook.ExecOp
	var mu sync.Mutex
	var trainDone bool
	var trainErr error

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
			return
		}

		coordinatorURL = request.CoordinatorURL
		opImpl := skyhook.GetExecOpImpl(request.Node.Op)
		var err error
		execOp, err = opImpl.Prepare(coordinatorURL, request.Node, request.OutputDatasets)
		if err != nil {
			panic(err)
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
			return
		}

		err := execOp.Apply(request.Task)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
	})

	http.HandleFunc("/train/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(404)
			return
		}

		var request skyhook.TrainBeginRequest
		if err := skyhook.ParseJsonRequest(w, r, &request); err != nil {
			return
		}

		coordinatorURL = request.CoordinatorURL

		op := skyhook.GetTrainOp(request.Node.Op)
		go func() {
			err := op.Train(coordinatorURL, request.Node)
			mu.Lock()
			trainDone = true
			trainErr = err
			mu.Unlock()
		}()

		skyhook.JsonResponse(w, skyhook.TrainBeginResponse{})
	})

	http.HandleFunc("/train/poll", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		done := trainDone
		err := trainErr
		mu.Unlock()
		var errorStr string
		if err != nil {
			errorStr = err.Error()
		}
		skyhook.JsonResponse(w, skyhook.TrainPollResponse{
			Done: done,
			Error: errorStr,
		})
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
