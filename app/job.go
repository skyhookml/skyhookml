package app

import (
	"github.com/skyhookml/skyhookml/skyhook"

	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
)

type DBJob struct {skyhook.Job}

const JobQuery = "SELECT id, name, type, op, metadata, start_time, done, error FROM jobs"

func jobListHelper(rows *Rows) []*DBJob {
	jobs := []*DBJob{}
	for rows.Next() {
		var j DBJob
		rows.Scan(&j.ID, &j.Name, &j.Type, &j.Op, &j.Metadata, &j.StartTime, &j.Done, &j.Error)
		jobs = append(jobs, &j)
	}
	return jobs
}

func ListJobs() []*DBJob {
	rows := db.Query(JobQuery + " ORDER BY id DESC")
	return jobListHelper(rows)
}

func GetJob(id int) *DBJob {
	rows := db.Query(JobQuery + " WHERE id = ?", id)
	jobs := jobListHelper(rows)
	if len(jobs) == 1 {
		return jobs[0]
	} else {
		return nil
	}
}

func NewJob(name string, t string, op string, metadata string) *DBJob {
	res := db.Exec(
		"INSERT INTO jobs (name, type, op, metadata, start_time) VALUES (?, ?, ?, ?, datetime('now'))",
		name, t, op, metadata,
	)
	return GetJob(res.LastInsertId())
}

func (j *DBJob) UpdateState(state string) {
	db.Exec("UPDATE jobs SET state = ? WHERE id = ?", state, j.ID)
}

func (j *DBJob) UpdateMetadata(metadata string) {
	db.Exec("UPDATE jobs SET metadata = ? WHERE id = ?", metadata, j.ID)
}

func (j *DBJob) GetState() string {
	var state string
	db.QueryRow("SELECT state FROM jobs WHERE id = ?", j.ID).Scan(&state)
	return state
}

var runningJobs = make(map[int]skyhook.JobOp)
var jobMu sync.Mutex

func (j *DBJob) AttachOp(op skyhook.JobOp) {
	jobMu.Lock()
	runningJobs[j.ID] = op
	jobMu.Unlock()
}

func (j *DBJob) SetDone(error string) {
	db.Exec("UPDATE jobs SET done = 1, error = ? WHERE id = ?", error, j.ID)
}

// A JobOp that wraps a TailOp for console, plus arbitrary number of other JobOps.
// It also provides functionality for stopping via mutex/condition.
type AppJobOp struct {
	Job *DBJob
	TailOp *skyhook.TailJobOp

	WrappedJobOps map[string]skyhook.JobOp
	LastWrappedDatas map[string]string

	// stopping support
	Stopping bool
	Stopped bool

	// function to call when stopping a job
	// we also call this on SetDone
	CleanupFunc func()

	mu sync.Mutex
	cond *sync.Cond
}

type AppJobState struct {
	Lines []string
	Datas map[string]string
}

func (op *AppJobOp) Encode() string {
	op.mu.Lock()
	defer op.mu.Unlock()
	// we only need to compute LastWrappedDatas if it isn't set yet
	if op.LastWrappedDatas == nil {
		op.LastWrappedDatas = make(map[string]string)
		for name, wrapped := range op.WrappedJobOps {
			op.LastWrappedDatas[name] = wrapped.Encode()
		}
	}
	state := AppJobState{
		Lines: op.TailOp.Lines,
		Datas: op.LastWrappedDatas,
	}
	return string(skyhook.JsonMarshal(state))
}

func (op *AppJobOp) Update(lines []string) {
	op.mu.Lock()
	defer op.mu.Unlock()

	op.TailOp.Update(lines)
	if op.LastWrappedDatas == nil {
		op.LastWrappedDatas = make(map[string]string)
	}
	for name, wrapped := range op.WrappedJobOps {
		wrapped.Update(lines)
		op.LastWrappedDatas[name] = wrapped.Encode()
	}
}

func (op *AppJobOp) SetCleanupFunc(f func()) {
	op.mu.Lock()
	op.CleanupFunc = f
	op.mu.Unlock()
}

// Run cleanup func and reset it.
// Caller must have the lock.
func (op *AppJobOp) cleanup() {
	if op.CleanupFunc != nil {
		op.CleanupFunc()
	}
	op.CleanupFunc = nil
}

func (op *AppJobOp) Cleanup() {
	op.mu.Lock()
	op.cleanup()
	op.mu.Unlock()
}

// Handles ending the job so that the caller doesn't need to call Job.SetDone directly.
func (op *AppJobOp) SetDone(err string) {
	op.mu.Lock()
	op.cleanup()

	op.Stopped = true
	if op.cond != nil {
		op.cond.Broadcast()
	}
	op.mu.Unlock()

	// make sure job state reflects the latest updates
	if err == "" {
		op.Update([]string{"Job completed successfully."})
	} else {
		op.Update([]string{"Job exiting with error: " + err})
	}
	op.Job.UpdateState(op.Encode())
	op.Job.SetDone(err)
}

func (op *AppJobOp) IsStopping() bool {
	op.mu.Lock()
	defer op.mu.Unlock()
	return op.Stopping
}

func (op *AppJobOp) Stop() error {
	op.mu.Lock()
	// Apply cleanup functions if needed, to interrupt the job.
	op.cleanup()
	// Set stopping and wait for the job thread to stop.
	// Job thread will notify on cond when it's stopped.
	op.Stopping = true
	if op.cond == nil {
		op.cond = sync.NewCond(&op.mu)
	}
	op.TailOp.Update([]string{"TERMINATING JOB: user requested to stop this job."})
	for !op.Stopped {
		op.cond.Wait()
	}
	op.mu.Unlock()
	return nil
}

type ProgressJobOp struct {
	Completed int
	Total int
	mu sync.Mutex
}

func (op *ProgressJobOp) Encode() string {
	op.mu.Lock()
	defer op.mu.Unlock()
	// compute progress
	// we check Total>0 because Total might not be set yet
	var progress int = 0
	if op.Total > 0 {
		progress = op.Completed*100/op.Total
	}
	return string(skyhook.JsonMarshal(progress))
}

func (op *ProgressJobOp) SetTotal(total int) {
	op.mu.Lock()
	op.Total = total
	op.Completed = 0
	op.mu.Unlock()
}

// Set the progress to the specified percentage.
// Returns true if the percent was updated.
func (op *ProgressJobOp) SetProgressPercent(percent int) bool {
	op.mu.Lock()
	defer op.mu.Unlock()
	if op.Total == 100 && op.Completed == percent {
		return false
	}
	op.Total = 100
	op.Completed = percent
	return true
}

func (op *ProgressJobOp) Increment() {
	op.mu.Lock()
	op.Completed++
	op.mu.Unlock()
}

func (op *ProgressJobOp) Update(lines []string) {}
func (op *ProgressJobOp) Stop() error { return nil }

func init() {
	Router.HandleFunc("/worker/job-update", func(w http.ResponseWriter, r *http.Request) {
		var request skyhook.JobUpdate
		if err := skyhook.ParseJsonRequest(w, r, &request); err != nil {
			return
		}
		jobMu.Lock()
		job := runningJobs[request.JobID]
		jobMu.Unlock()
		job.Update(request.Lines)
		state := job.Encode()
		(&DBJob{Job: skyhook.Job{ID: request.JobID}}).UpdateState(state)
	})

	Router.HandleFunc("/jobs", func(w http.ResponseWriter, r *http.Request) {
		skyhook.JsonResponse(w, ListJobs())
	}).Methods("GET")

	Router.HandleFunc("/jobs/{job_id}", func(w http.ResponseWriter, r *http.Request) {
		jobID := skyhook.ParseInt(mux.Vars(r)["job_id"])
		job := GetJob(jobID)
		if job == nil {
			http.Error(w, "no such job", 404)
			return
		}
		skyhook.JsonResponse(w, job)
	}).Methods("GET")

	Router.HandleFunc("/jobs/{job_id}/state", func(w http.ResponseWriter, r *http.Request) {
		jobID := skyhook.ParseInt(mux.Vars(r)["job_id"])
		job := GetJob(jobID)
		if job == nil {
			http.Error(w, "no such job", 404)
			return
		}

		if !job.Done {
			jobMu.Lock()
			jobOp := runningJobs[jobID]
			jobMu.Unlock()
			if jobOp != nil {
				state := jobOp.Encode()
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(state))
				return
			}
		}

		state := job.GetState()
		if state == "" {
			state = "null"
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(state))
	}).Methods("POST")

	Router.HandleFunc("/jobs/{job_id}/stop", func(w http.ResponseWriter, r *http.Request) {
		jobID := skyhook.ParseInt(mux.Vars(r)["job_id"])
		jobMu.Lock()
		job := runningJobs[jobID]
		jobMu.Unlock()
		if job == nil {
			http.Error(w, "no such running job", 404)
			return
		}
		err := job.Stop()
		if err != nil {
			log.Printf("[job-stop] error stopping job: %v", err)
			http.Error(w, fmt.Sprintf("error stopping job: %v", err), 404)
			return
		}
	}).Methods("POST")
}
