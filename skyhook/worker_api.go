package skyhook

// worker->coordinator
type WorkerInitRequest struct {
	BaseURL string
}

// either coordinator->worker or worker->container
type ExecBeginRequest struct {
	Node Runnable
	JobID *int

	// only set for worker->container
	CoordinatorURL string
}

type ExecBeginResponse struct {
	Parallelism int

	// filled in by worker for response back to coordinator
	UUID string
	BaseURL string
}

type JobUpdate struct {
	JobID int
	Lines []string
}

type EndRequest struct {
	UUID string
}

// coordinator->container
type ExecTaskRequest struct {
	Task ExecTask
}
