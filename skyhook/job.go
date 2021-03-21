package skyhook

import (
	"fmt"
	"time"
)

type Job struct {
	ID int
	Name string
	Type string
	Op string
	Metadata string
	StartTime time.Time
	State string

	// If the job succeeds, Done=true and Error="".
	// If it fails, then Done=true and Error is set.
	// If Done=false it implies the job is still running.
	Done bool
	Error string
}

type JobOp interface {
	// Update the job given the newly received lines from the job output.
	Update(lines []string)
	// Encode the current job state
	Encode() string
	// Stop this job.
	Stop() error
}

// JobOp implementation that just keeps the latest 1000 lines of output
// This is used as a helper not as an actual JobOp -- since it can't stop the job.
// It is not thread-safe.
const TailJobOpNumLines int = 1000
type TailJobOp struct {
	Lines []string
	numLines int
}
func (op *TailJobOp) Update(lines []string) {
	if op.numLines == 0 {
		op.numLines = TailJobOpNumLines
	}

	// add lines to op.Lines until we exceed DefaultJobOpNumLines
	if len(op.Lines) < op.numLines {
		n := len(lines)
		if n > op.numLines - len(op.Lines) {
			n = op.numLines - len(op.Lines)
		}
		op.Lines = append(op.Lines, lines[0:n]...)
		lines = lines[n:]
	}

	// now that op.Lines is full, add as many as we can
	if len(lines) > op.numLines {
		lines = lines[len(lines)-op.numLines:]
	}
	if len(lines) > 0 {
		// shift to the left
		copy(op.Lines[0:], op.Lines[len(lines):])
		// and then insert
		copy(op.Lines[len(op.Lines)-len(lines):], lines)
	}
}
func (op *TailJobOp) Encode() string {
	return string(JsonMarshal(op.Lines))
}
func  (op *TailJobOp) Stop() error {
	panic(fmt.Errorf("Stop should never be called on TailJobOp"))
}

type ModelJobState struct {
	TrainLoss []float64
	ValLoss []float64
}
