package skyhook

// worker->coordinator
type WorkerInitRequest struct {
	BaseURL string
}

// either coordinator->worker or worker->container
type ExecBeginRequest struct {
	Node ExecNode
	OutputDatasets []Dataset

	// only set for worker->container
	CoordinatorURL string
}

type ExecBeginResponse struct {
	Parallelism int

	// filled in by worker for response back to coordinator
	UUID string
	BaseURL string
}

type EndRequest struct {
	UUID string
}

// coordinator->container
type ExecTaskRequest struct {
	Task ExecTask
}

type TrainBeginRequest struct {
	Node TrainNode

	// only set for worker->container
	CoordinatorURL string
}

type TrainBeginResponse struct {
	// filled in by worker for response back to coordinator
	UUID string
	BaseURL string
}

type TrainPollResponse struct {
	Done bool
	Error string
}
