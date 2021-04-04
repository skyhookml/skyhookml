package skyhook

// coordinator->worker
// request creation of a new container
type ContainerRequest struct {
	Node Runnable
	JobID *int
	CoordinatorURL string
}
type ContainerResponse struct {
	// request/container UUID
	UUID string
}

// coordinator->worker
// sent repeatedly to get status of the ContainerRequest
type StatusRequest struct {
	UUID string
}
type StatusResponse struct {
	// whether the container has been provisioned
	Ready bool
	// if not Ready, some message e.g. position in queue
	Message string
	// if Ready, forwarded ExecBeginResponse
	ExecBeginResponse ExecBeginResponse
	// if Ready, base URL where container can be accessed
	BaseURL string
}

// worker->container
type ExecBeginRequest struct {
	Node Runnable
	JobID *int
	CoordinatorURL string
}
type ExecBeginResponse struct {
	Parallelism int
}

type JobUpdate struct {
	JobID int
	Lines []string
}

// release a container
type EndRequest struct {
	UUID string
}

// coordinator->container
type ExecTaskRequest struct {
	Task ExecTask
}
