package app

import (
	"../skyhook"

	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
)

type DBJob struct {skyhook.Job}

const JobFastQuery = "SELECT id, name, type, op, metadata, start_time, done, error, '' FROM jobs"
const JobQuery = "SELECT id, name, type, op, metadata, start_time, done, error, state FROM jobs"

func jobListHelper(rows *Rows) []*DBJob {
	jobs := []*DBJob{}
	for rows.Next() {
		var j DBJob
		rows.Scan(&j.ID, &j.Name, &j.Type, &j.Op, &j.Metadata, &j.StartTime, &j.Done, &j.Error, &j.State)
		jobs = append(jobs, &j)
	}
	return jobs
}

func ListJobs() []*DBJob {
	rows := db.Query(JobFastQuery + " ORDER BY id DESC")
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
	j.State = state
	db.Exec("UPDATE jobs SET state = ? WHERE id = ?", state, j.ID)
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
