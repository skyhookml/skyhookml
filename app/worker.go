package app

import (
	"github.com/skyhookml/skyhookml/skyhook"

	"fmt"
	"log"
	"sync"
)

// Lock access to the backend worker or pool.
// In the future (after worker_pool has more capabilities), we should just pass
// requests if the configured endpoint is a pool.

var workerMu sync.Mutex
var workerCond *sync.Cond
var workerInUse bool

// Acquire worker and return nil.
// Or returns error if interrupted (i.e. job terminated by user).
func AcquireWorker(jobOp *AppJobOp) error {
	stop := false

	jobOp.SetCleanupFunc(func() {
		workerMu.Lock()
		stop = true
		workerCond.Broadcast()
		workerMu.Unlock()
	})

	workerMu.Lock()
	defer workerMu.Unlock()
	for workerInUse && !stop {
		workerCond.Wait()
	}
	if stop {
		return fmt.Errorf("job terminated while acquiring worker")
	}
	jobOp.SetCleanupFunc(nil)
	workerInUse = true
	return nil
}

func ReleaseWorker() {
	workerMu.Lock()
	workerInUse = false
	workerCond.Broadcast()
	workerMu.Unlock()
}

func init() {
	workerCond = sync.NewCond(&workerMu)
}

// Allocate a container on the worker.
// Caller is responsible for acquiring worker.
type ContainerInfo struct {
	UUID string
	BaseURL string
	Parallelism int
}
func AcquireContainer(node skyhook.Runnable, jobOp *AppJobOp) (ContainerInfo, error) {
	println := func(s string) {
		jobOp.Update([]string{s})
		log.Printf("[acquire-container job-%d] %s", jobOp.Job.ID, s)
	}

	println("Acquiring container")
	containerRequest := skyhook.ContainerRequest{
		Node: node,
		JobID: &jobOp.Job.ID,
		CoordinatorURL: Config.CoordinatorURL,
		InstanceID: Config.InstanceID,
	}
	var containerResponse skyhook.ContainerResponse
	err := skyhook.JsonPost(Config.WorkerURL, "/container/request", containerRequest, &containerResponse)
	if err != nil {
		return ContainerInfo{}, err
	}
	uuid := containerResponse.UUID

	// Wait for container to become available.
	// We poll the container status in a separate goroutine, and save the response
	// to a shared variable controlled by mutex.
	// This way, we can return from AcquireContainer early if the job is terminated.
	// In case of termination, stopped will be set true, and the goroutine should
	// release the container as soon as it is ready.
	var response struct {
		info ContainerInfo
		err error
		done bool
	}
	stopped := false // will be set true if job is terminated
	var mu sync.Mutex
	cond := sync.NewCond(&mu)
	go func() {
		info, err := func() (ContainerInfo, error) {
			for {
				request := skyhook.StatusRequest{UUID: uuid}
				var response skyhook.StatusResponse
				err := skyhook.JsonPost(Config.WorkerURL, "/container/status", request, &response)
				if err != nil {
					return ContainerInfo{}, err
				}
				if !response.Ready {
					println(fmt.Sprintf("... still waiting: %s", response.Message))
					continue
				}
				return ContainerInfo{
					UUID: uuid,
					BaseURL: response.BaseURL,
					Parallelism: response.ExecBeginResponse.Parallelism,
				}, nil
			}
		}()

		mu.Lock()
		if stopped {
			// The job was terminated so nobody is waiting for this response anymore.
			// We release the container immediately.
			err := skyhook.JsonPost(Config.WorkerURL, "/container/end", skyhook.EndRequest{info.UUID}, nil)
			if err != nil {
				log.Printf("error releasing container %s: %v", info.UUID, err)
			}
		} else {
			response.info = info
			response.err = err
			response.done = true
			cond.Broadcast()
		}
		mu.Unlock()
	}()

	jobOp.SetCleanupFunc(func() {
		mu.Lock()
		stopped = true
		cond.Broadcast()
		mu.Unlock()
	})
	mu.Lock()
	defer mu.Unlock()
	for !stopped && !response.done {
		cond.Wait()
	}
	if !response.done {
		return ContainerInfo{}, fmt.Errorf("job terminated while waiting for container")
	}
	return response.info, response.err
}
