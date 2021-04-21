package app

import (
	"github.com/skyhookml/skyhookml/skyhook"

	"fmt"
	"sync"
)

// A JobOp for running multiple ExecNodes.
type MultiExecJobOp struct {
	mu sync.Mutex
	Job *DBJob

	// current wrapped job (current ExecJob)
	CurJob *skyhook.Job

	// current execution plan
	// the field can change but the slice itself must not
	Plan []*skyhook.VirtualNode

	// which index in the plan are we executing next (or right now)?
	PlanIndex int
}

type MultiExecJobState struct {
	CurJob *skyhook.Job
	Plan []*skyhook.VirtualNode
	PlanIndex int
}

func (op *MultiExecJobOp) Encode() string {
	op.mu.Lock()
	defer op.mu.Unlock()
	return string(skyhook.JsonMarshal(MultiExecJobState{
		CurJob: op.CurJob,
		Plan: op.Plan,
		PlanIndex: op.PlanIndex,
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
func (op *MultiExecJobOp) ChangePlan(plan []*skyhook.VirtualNode, planIndex int) {
	op.mu.Lock()
	op.Plan = plan
	op.PlanIndex = planIndex
	op.mu.Unlock()
}

func (op *MultiExecJobOp) ChangeJob(job skyhook.Job) {
	op.mu.Lock()
	op.CurJob = &job
	op.mu.Unlock()
}

// Get a []*skyhook.VirtualNode plan based on current execution graph and related state.
func (op *MultiExecJobOp) SetPlanFromGraph(graph skyhook.ExecutionGraph, ready map[skyhook.GraphID]map[string]*DBDataset, needed map[skyhook.GraphID]skyhook.Node, cur *skyhook.VirtualNode) {
	var plan []*skyhook.VirtualNode
	addGraphID := func(gid skyhook.GraphID) {
		vnode, ok := graph[gid].(*skyhook.VirtualNode)
		if !ok {
			return
		}
		if vnode == cur {
			return
		}
		plan = append(plan, vnode)
	}
	for gid := range ready {
		addGraphID(gid)
	}
	planIndex := len(plan)
	if cur != nil {
		plan = append(plan, cur)
	}
	for gid := range needed {
		addGraphID(gid)
	}
	op.ChangePlan(plan, planIndex)
}
