package main

import (
	"github.com/skyhookml/skyhookml/skyhook"
	gouuid "github.com/google/uuid"

	_ "github.com/skyhookml/skyhookml/ops"

	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var coordinatorURL string

func main() {
	if len(os.Args) < 4 {
		fmt.Println("usage: ./worker [external IP] [port] [skyhook URL]")
		fmt.Println("example: ./worker localhost 8081 http://localhost:8080")
		return
	}
	myIP := os.Args[1]
	myPort := skyhook.ParseInt(os.Args[2])
	coordinatorURL = os.Args[3]

	mode := "docker"
	if len(os.Args) >= 5 {
		mode = os.Args[4]
	}

	workingDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	type Cmd struct {
		Cmd *exec.Cmd
		Port int
		BaseURL string
	}
	containers := make(map[string]*Cmd)
	ports := []int{8100, 8101, 8102, 8103}
	var mu sync.Mutex

	getPort := func() int {
		usedSet := make(map[int]bool)
		for _, cmd := range containers {
			usedSet[cmd.Port] = true
		}
		for _, port := range ports {
			if !usedSet[port] {
				return port
			}
		}
		panic(fmt.Errorf("no available port"))
	}

	// Returns (container base URL, uuid, error)
	startContainer := func(imageName string, jobID *int) (string, string, error) {
		uuid := gouuid.New().String()

		mu.Lock()
		containerPort := getPort()
		containerBaseURL := fmt.Sprintf("http://localhost:%d", containerPort)
		containers[uuid] = &Cmd{
			Cmd: nil,
			Port: containerPort,
			BaseURL: containerBaseURL,
		}
		mu.Unlock()

		var cmd *exec.Cmd
		if mode == "docker" {
			cmd = exec.Command(
				"docker", "run",
				"--mount", fmt.Sprintf("\"src=%s\",target=/usr/src/app/skyhook/items,type=bind", filepath.Join(workingDir, "items")),
				"--mount", fmt.Sprintf("\"src=%s\",target=/usr/src/app/skyhook/models,type=bind", filepath.Join(workingDir, "models")),
				"--gpus", "all",
				"-p", fmt.Sprintf("%d:8080", containerPort),
				"--name", uuid,
				imageName,
			)
		} else if mode == "process" {
			cmd = exec.Command(
				"go", "run", "cmd/container.go", fmt.Sprintf(":%d", containerPort),
			)
		}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			panic(err)
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			panic(err)
		}
		if err := cmd.Start(); err != nil {
			panic(err)
		}
		log.Printf("[machine] container %s started", uuid)

		mu.Lock()
		containers[uuid].Cmd = cmd
		mu.Unlock()

		// wait for container to be ready
		stdoutRd := bufio.NewReader(stdout)
		stdoutRd.ReadString('\n') // ignore error here since it'll be caught by readContainerOutput
		// read stdout/stderr, and if JobID is set then pass the output lines to the coordinator
		readContainerOutput(uuid, stdoutRd, bufio.NewReader(stderr), jobID)

		return containerBaseURL, uuid, nil
	}

	http.HandleFunc("/exec/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(404)
			return
		}

		var request skyhook.ExecBeginRequest
		if err := skyhook.ParseJsonRequest(w, r, &request); err != nil {
			return
		}

		imageName, err := request.Node.GetOp().GetImageName(request.Node)
		if err != nil {
			panic(err)
		}

		containerBaseURL, uuid, err := startContainer(imageName, request.JobID)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		request.CoordinatorURL = coordinatorURL
		var response skyhook.ExecBeginResponse
		err = skyhook.JsonPost(containerBaseURL, "/exec/start", request, &response)
		if err != nil {
			panic(err)
		}

		response.UUID = uuid
		response.BaseURL = containerBaseURL
		skyhook.JsonResponse(w, response)
	})

	http.HandleFunc("/end", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(404)
			return
		}

		var request skyhook.EndRequest
		if err := skyhook.ParseJsonRequest(w, r, &request); err != nil {
			return
		}
		uuid := request.UUID

		mu.Lock()
		cmd := containers[uuid]
		delete(containers, uuid)
		mu.Unlock()

		if cmd == nil {
			return
		}

		if mode == "docker" {
			err := exec.Command("docker", "rm", "--force", uuid).Run()
			if err != nil {
				panic(err)
			}
			cmd.Cmd.Wait()
		} else if mode == "process" {
			skyhook.JsonPost(cmd.BaseURL, "/exit", nil, nil)
			if err := cmd.Cmd.Wait(); err != nil {
				panic(err)
			}
		}

		log.Printf("[machine] container %s stopped", uuid)
	})

	// register with the coordinator
	initRequest := skyhook.WorkerInitRequest{
		BaseURL: fmt.Sprintf("http://%s:%d", myIP, myPort),
	}
	err = skyhook.JsonPost(coordinatorURL, "/worker/init", initRequest, nil)
	if err != nil {
		panic(err)
	}

	log.Printf("starting on :%d", myPort)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", myPort), nil); err != nil {
		panic(err)
	}
}

func readContainerOutput(uuid string, stdout *bufio.Reader, stderr *bufio.Reader, jobID *int) {
	// Read lines from stdout and stderr simultaneously, printing to our output
	// Every second, accumulate the lines (if any) and forward to coordinator (if jobID is set).
	var mu sync.Mutex
	var pending int = 0
	var lines []string
	readLines := func(rd *bufio.Reader) {
		pending++
		go func() {
			for {
				line, err := rd.ReadString('\n')
				if err != nil {
					break
				}
				log.Printf("[container %s] %s", uuid, line)
				if jobID != nil {
					mu.Lock()
					lines = append(lines, strings.Trim(line, "\n\r"))
					mu.Unlock()
				}
			}
			mu.Lock()
			pending--
			mu.Unlock()
		}()
	}
	readLines(stdout)
	readLines(stderr)
	if jobID != nil {
		go func() {
			for {
				time.Sleep(time.Second)
				mu.Lock()
				curLines := lines
				lines = nil
				curPending := pending
				mu.Unlock()

				if len(curLines) == 0 && curPending == 0 {
					break
				}
				if len(curLines) == 0 {
					continue
				}

				// forward to coordinator
				request := skyhook.JobUpdate{
					JobID: *jobID,
					Lines: curLines,
				}
				err := skyhook.JsonPost(coordinatorURL, "/worker/job-update", request, nil)
				if err != nil {
					panic(err)
				}
			}
		}()
	}
}
