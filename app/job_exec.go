package app

import (
	"../skyhook"

	"fmt"
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
	LastMetadata interface{}
}

type ExecJobState struct {
	Progress int
	Lines []string
	Metadata interface{}
}

func (op *ExecJobOp) Encode() interface{} {
	return ExecJobState{
		Progress: op.CompletedTasks * 100 / op.NumTasks,
		Lines: op.TailOp.Lines,
		Metadata: op.LastMetadata,
	}
}

func (op *ExecJobOp) Update(lines []string) interface{} {
	op.TailOp.Update(lines)
	if op.NodeJobOp != nil {
		op.LastMetadata = op.NodeJobOp.Update(lines)
	}
	return op.Encode()
}

func (op *ExecJobOp) Completed(key string) {
	op.CompletedTasks++
	op.TailOp.Update([]string{fmt.Sprintf("finished applying on key [%s]", key)})

	if time.Now().Sub(op.lastTime) > ExecJobOpUpdateDelay {
		state := string(skyhook.JsonMarshal(op.Encode()))
		op.Job.UpdateState(state)
		op.lastTime = time.Now()
	}
}
