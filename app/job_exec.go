package app

import (
	"github.com/skyhookml/skyhookml/skyhook"

	"fmt"
	"sync"
	"time"
)

const ExecJobOpUpdateDelay time.Duration = time.Second

// A JobOp for an ExecNode.
// - track the percentage of tasks completed
// - use skyhook.TailJobOp to track the last lines of console output
// - add a console line whenever a task completes
type ExecJobOp struct {
	Job *DBJob
	NumTasks int
	CompletedTasks int
	TailOp *skyhook.TailJobOp

	// last time we updated the db based on completed task
	lastTime time.Time

	// JobOp provided by the ExecNode, if any
	NodeJobOp skyhook.JobOp
	LastMetadata string

	// support for terminating a job
	Stopping bool
	Stopped bool

	// functions we need to call when stopping a job or on SetDone
	CleanupFuncs []func()

	mu sync.Mutex
	cond *sync.Cond
}

type ExecJobState struct {
	Progress int
	Lines []string
	Metadata string
}

// Encodes the state.
// Caller must have the lock.
func (op *ExecJobOp) encode() string {
	state := ExecJobState{
		Progress: op.CompletedTasks * 100 / op.NumTasks,
		Lines: op.TailOp.Lines,
		Metadata: op.LastMetadata,
	}
	return string(skyhook.JsonMarshal(state))
}

func (op *ExecJobOp) Encode() string {
	op.mu.Lock()
	defer op.mu.Unlock()
	return op.encode()
}

func (op *ExecJobOp) Update(lines []string) {
	op.mu.Lock()
	defer op.mu.Unlock()

	op.TailOp.Update(lines)
	if op.NodeJobOp != nil {
		op.NodeJobOp.Update(lines)
		op.LastMetadata = op.NodeJobOp.Encode()
	}
}

func (op *ExecJobOp) Completed(key string) {
	op.mu.Lock()
	defer op.mu.Unlock()

	op.CompletedTasks++
	op.TailOp.Update([]string{fmt.Sprintf("finished applying on key [%s]", key)})

	if time.Now().Sub(op.lastTime) > ExecJobOpUpdateDelay {
		op.Job.UpdateState(op.encode())
		op.lastTime = time.Now()
	}
}

func (op *ExecJobOp) AddCleanupFunc(f func()) {
	op.mu.Lock()
	op.CleanupFuncs = append(op.CleanupFuncs, f)
	op.mu.Unlock()
}

// Run cleanup funcs and reset them.
// Caller must have the lock.
func (op *ExecJobOp) cleanup() {
	for _, f := range op.CleanupFuncs {
		f()
	}
	op.CleanupFuncs = nil
}

// Always called when Run() exits the function call.
func (op *ExecJobOp) SetDone(err string) {
	op.mu.Lock()
	if err == "" {
		op.TailOp.Update([]string{"success!"})
	} else {
		op.TailOp.Update([]string{"exiting with error: " + err})
	}

	// make sure job state reflects the latest updates
	op.Job.UpdateState(op.encode())
	op.Job.SetDone(err)

	op.cleanup()
	op.Stopped = true
	op.cond.Broadcast()
	op.mu.Unlock()
}

func (op *ExecJobOp) IsStopping() bool {
	op.mu.Lock()
	defer op.mu.Unlock()
	return op.Stopping
}

func (op *ExecJobOp) Stop() error {
	op.mu.Lock()
	op.cleanup()
	op.Stopping = true
	op.TailOp.Update([]string{"!!! user requested to stop this job !!!"})
	for !op.Stopped {
		op.cond.Wait()
	}
	op.mu.Unlock()
	return nil
}
