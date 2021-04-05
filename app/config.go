package app

// Global config object, set by main.go
var Config struct {
	// URL where main program can be reached.
	// This is used when telling workers which coordinator a request came from,
	// so that we can get back responses or serve any API calls worker needs to make.
	CoordinatorURL string
	// URL where the worker is, which runs as a separate program.
	// Can also point to a worker pool.
	WorkerURL string
	// Optional instance ID.
	// If set, the worker should launch container in a subdirectory with this name.
	InstanceID string
}
