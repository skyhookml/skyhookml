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

func AcquireWorker() {
	workerMu.Lock()
	defer workerMu.Unlock()
	for workerInUse {
		workerCond.Wait()
	}
	workerInUse = true
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
	var info ContainerInfo
	containerRequest := skyhook.ContainerRequest{
		Node: node,
		JobID: &jobOp.Job.ID,
		CoordinatorURL: Config.CoordinatorURL,
	}
	var containerResponse skyhook.ContainerResponse
	err := skyhook.JsonPost(Config.WorkerURL, "/container/request", containerRequest, &containerResponse)
	if err != nil {
		return info, err
	}
	info.UUID = containerResponse.UUID

	// wait for container to become available
	for {
		request := skyhook.StatusRequest{UUID: info.UUID}
		var response skyhook.StatusResponse
		err := skyhook.JsonPost(Config.WorkerURL, "/container/status", request, &response)
		if err != nil {
			return info, err
		}
		if !response.Ready {
			println(fmt.Sprintf("... still waiting: %s", response.Message))
			continue
		}
		info.BaseURL = response.BaseURL
		info.Parallelism = response.ExecBeginResponse.Parallelism
		break
	}

	return info, nil
}
