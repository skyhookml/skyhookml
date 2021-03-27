package app

import (
	"github.com/skyhookml/skyhookml/skyhook"

	"fmt"
	"sync"
)

// A JobOp for running multiple ExecNodes.
type MultiExecJobOp struct {
	mu sync.Mutex

	// current wrapped job (current ExecJob)
	CurJob *skyhook.Job

	// current execution plan
	// the field can change but the slice itself must not
	Plan []*skyhook.VirtualNode
}

type MultiExecJobState struct {
	CurJob *skyhook.Job
	Plan []*skyhook.VirtualNode
}

func (op *MultiExecJobOp) Encode() string {
	op.mu.Lock()
	defer op.mu.Unlock()
	return string(skyhook.JsonMarshal(MultiExecJobState{
		CurJob: op.CurJob,
		Plan: op.Plan,
	}))
}

func (op *MultiExecJobOp) Update(lines []string) {
	panic(fmt.Errorf("Update should not be called on MultiExecJobOp"))
}
func (op *MultiExecJobOp) Stop() error {
	panic(fmt.Errorf("Stop should not be called on MultiExecJobOp"))
}

// Set the plan.
// The plan must be immutable.
func (op *MultiExecJobOp) ChangePlan(plan []*skyhook.VirtualNode) {
	op.mu.Lock()
	op.Plan = plan
	op.mu.Unlock()
}

func (op *MultiExecJobOp) ChangeJob(job skyhook.Job) {
	op.mu.Lock()
	op.CurJob = &job
	op.mu.Unlock()
}
