package main

import (
	"github.com/skyhookml/skyhookml/skyhook"
	gouuid "github.com/google/uuid"

	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Worker struct {
	URL string
	// UUID of currently allocated container, if any (otherwise empty)
	// currently we only allocate one container at a time per worker
	ContainerUUID string
	// UUID that we returned to coordinator
	AllocationUUID string
}

type Request struct {
	skyhook.ContainerRequest
	UUID string
}

// Store result of allocation after a request exits the queue.
// Either Error is set, or it is successful and response/baseURL set.
type AllocationResult struct {
	ExecBeginResponse skyhook.ExecBeginResponse
	BaseURL string
	Error error
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: ./worker_pool [port] [worker list]")
		fmt.Println("example: ./worker_pool 8081 http://1.2.3.4:8081,http://5.6.7.8:8081")
		return
	}
	myPort := skyhook.ParseInt(os.Args[1])
	workerURLs := strings.Split(os.Args[2], ",")

	// maintain state of the workers
	workers := make([]*Worker, len(workerURLs))
	for i, url := range workerURLs {
		workers[i] = &Worker{URL: url}
	}
	// maintain queue of container requests
	var q []*Request
	// result of completed (succeeded/failed) requests
	results := make(map[string]*AllocationResult)

	var mu sync.Mutex
	cond := sync.NewCond(&mu)

	// wakeup cond so that /container/status pollers can timeout
	go func() {
		for {
			time.Sleep(5*time.Second)
			cond.Broadcast()
		}
	}()

	// process requests
	go func() {
		// helper function checking if there's an available worker
		// caller must have the lock
		getAvailableWorker := func() *Worker {
			for _, worker := range workers {
				if worker.ContainerUUID != "" {
					continue
				}
				return worker
			}
			return nil
		}

		for {
			// wait for a request
			mu.Lock()
			for len(q) == 0 {
				cond.Wait()
			}
			req := q[0]
			log.Printf("[queue] got request %s, waiting for a worker", req.UUID)

			// wait for a worker
			for getAvailableWorker() == nil {
				cond.Wait()
			}
			worker := getAvailableWorker()
			log.Printf("[req %s] got candidate worker at %s", req.UUID, worker.URL)
			mu.Unlock()

			setError := func(err error) {
				log.Printf("[req %s] error allocating on worker %s: %v", req.UUID, worker.URL, err)
				mu.Lock()
				n := copy(q[0:], q[1:])
				q = q[0:n]
				results[req.UUID] = &AllocationResult{Error: err}
				mu.Unlock()
			}

			// forward the ContainerRequest
			var containerResponse skyhook.ContainerResponse
			err := skyhook.JsonPost(worker.URL, "/container/request", req.ContainerRequest, &containerResponse)
			if err != nil {
				setError(err)
				continue
			}

			// Call /container/request.
			// This should always respond with a final status, i.e., either
			// ready=true or Error is non-nil.
			// Only pool (us) responds with pending update.
			statusRequest := skyhook.StatusRequest{UUID: containerResponse.UUID}
			var statusResponse skyhook.StatusResponse
			err = skyhook.JsonPost(worker.URL, "/container/status", statusRequest, &statusResponse)
			if err != nil {
				setError(err)
				continue
			}

			if !statusResponse.Ready {
				panic(fmt.Errorf("got status response from worker with ready=false"))
			}

			log.Printf("[req %s] successfully allocated on worker %s at %s", req.UUID, worker.URL, statusResponse.BaseURL)
			mu.Lock()
			n := copy(q[0:], q[1:])
			q = q[0:n]
			results[req.UUID] = &AllocationResult{
				ExecBeginResponse: statusResponse.ExecBeginResponse,
				BaseURL: statusResponse.BaseURL,
			}
			worker.ContainerUUID = containerResponse.UUID
			worker.AllocationUUID = req.UUID
			cond.Broadcast()
			mu.Unlock()

		}
	}()

	http.HandleFunc("/container/request", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(404)
			return
		}
		var request skyhook.ContainerRequest
		if err := skyhook.ParseJsonRequest(w, r, &request); err != nil {
			return
		}
		uuid := gouuid.New().String()
		log.Printf("[req %s] append new request to the queue, node_op=%s coordinator=%s", uuid, request.Node.Op, request.CoordinatorURL)
		mu.Lock()
		q = append(q, &Request{
			ContainerRequest: request,
			UUID: uuid,
		})
		cond.Broadcast()
		mu.Unlock()
		skyhook.JsonResponse(w, skyhook.ContainerResponse{uuid})
	})

	http.HandleFunc("/container/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(404)
			return
		}
		var request skyhook.StatusRequest
		if err := skyhook.ParseJsonRequest(w, r, &request); err != nil {
			return
		}
		var response *skyhook.StatusResponse
		var err error
		startTime := time.Now()
		mu.Lock()
		for {
			var indexInQueue int
			response, err, indexInQueue = func() (*skyhook.StatusResponse, error, int) {
				// is a result available?
				if results[request.UUID] != nil {
					res := results[request.UUID]
					if res.Error != nil {
						return nil, res.Error, 0
					} else {
						return &skyhook.StatusResponse{
							Ready: true,
							ExecBeginResponse: res.ExecBeginResponse,
							BaseURL: res.BaseURL,
						}, nil, 0
					}
				}

				// is it in the queue?
				for i, req := range q {
					if req.UUID != request.UUID {
						continue
					}
					return nil, nil, i
				}

				return nil, fmt.Errorf("UUID not found"), 0
			}()

			if response != nil || err != nil {
				break
			}

			// still in queue, see if we should timeout or keep waiting
			if time.Now().Sub(startTime) > 30*time.Second {
				response = &skyhook.StatusResponse{
					Message: fmt.Sprintf("position in queue is %d", indexInQueue),
				}
				break
			}

			cond.Wait()
		}
		mu.Unlock()
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		skyhook.JsonResponse(w, response)
	})

	http.HandleFunc("/container/end", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(404)
			return
		}
		var request skyhook.EndRequest
		if err := skyhook.ParseJsonRequest(w, r, &request); err != nil {
			return
		}
		uuid := request.UUID

		// check if any worker currently has this UUID allocated
		// also get the backend container UUID (pool assigns one UUID and worker assigns another)
		var worker *Worker
		var containerUUID string
		mu.Lock()
		for _, w := range workers {
			if w.AllocationUUID == uuid {
				worker = w
				containerUUID = worker.ContainerUUID
				break
			}
		}
		mu.Unlock()
		if worker == nil {
			return
		}

		log.Printf("[req %s] stopping container %s on worker %s", uuid, containerUUID, worker.URL)
		err := skyhook.JsonPost(worker.URL, "/container/end", skyhook.EndRequest{
			UUID: containerUUID,
		}, nil)
		if err != nil {
			log.Printf("[req %s] warning: error ending container on %s: %v", uuid, worker.URL, err)
			http.Error(w, err.Error(), 400)
			return
		}

		log.Printf("[req %s] stopped successfully, marking worker %s available", uuid, worker.URL)
		mu.Lock()
		worker.AllocationUUID = ""
		worker.ContainerUUID = ""
		mu.Unlock()
	})

	log.Printf("starting on :%d", myPort)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", myPort), nil); err != nil {
		panic(err)
	}
}
