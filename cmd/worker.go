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

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: ./worker [external IP] [port]")
		fmt.Println("example: ./worker localhost 8081")
		return
	}
	myIP := os.Args[1]
	myPort := skyhook.ParseInt(os.Args[2])

	mode := "docker"
	if len(os.Args) >= 4 {
		mode = os.Args[3]
	}

	workingDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	type Container struct {
		// Cmd, Port, and BaseURL are zero if container hasn't been provisioned yet,
		// or if there was an error provisioning it.
		Cmd *exec.Cmd
		Port int
		BaseURL string
		// Only set after container is ready.
		Ready bool
		ExecBeginResponse skyhook.ExecBeginResponse
		// Set if we have error creating the container
		Error error
	}
	containers := make(map[string]*Container)
	ports := []int{8100, 8101, 8102, 8103}
	var mu sync.Mutex
	cond := sync.NewCond(&mu)

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

	// Returns (container base URL, error)
	startContainer := func(uuid string, imageName string, jobID *int, coordinatorURL string) (string, error) {
		mu.Lock()
		containerPort := getPort()
		containerBaseURL := fmt.Sprintf("http://%s:%d", myIP, containerPort)
		containers[uuid].Port = containerPort
		containers[uuid].BaseURL = containerBaseURL
		mu.Unlock()

		var cmd *exec.Cmd
		if mode == "docker" {
			cmd = exec.Command(
				"docker", "run",
				"--mount", fmt.Sprintf("\"src=%s\",target=/usr/src/app/skyhook/data,type=bind", filepath.Join(workingDir, "data")),
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
		readContainerOutput(uuid, stdoutRd, bufio.NewReader(stderr), jobID, coordinatorURL)

		return containerBaseURL, nil
	}

	http.HandleFunc("/container/request", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(404)
			return
		}

		var request skyhook.ContainerRequest
		if err := skyhook.ParseJsonRequest(w, r, &request); err != nil {
			return
		}
		uuid := gouuid.New().String()
		mu.Lock()
		containers[uuid] = &Container{}
		mu.Unlock()
		skyhook.JsonResponse(w, skyhook.ContainerResponse{uuid})

		go func() {
			imageName, err := request.Node.GetOp().GetImageName(request.Node)
			if err != nil {
				panic(err)
			}

			baseURL, err := startContainer(uuid, imageName, request.JobID, request.CoordinatorURL)
			if err != nil {
				// don't really need to do anything since startContainer will set containers[uuid].Error
				return
			}

			execRequest := skyhook.ExecBeginRequest{
				Node: request.Node,
				JobID: request.JobID,
				CoordinatorURL: request.CoordinatorURL,
			}

			var response skyhook.ExecBeginResponse
			err = skyhook.JsonPost(baseURL, "/exec/start", execRequest, &response)
			if err != nil {
				panic(err)
			}

			mu.Lock()
			containers[uuid].ExecBeginResponse = response
			containers[uuid].Ready = true
			cond.Broadcast()
			mu.Unlock()
		}()
	})

	http.HandleFunc("/container/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(404)
			return
		}
		var request skyhook.StatusRequest
		if err := skyhook.ParseJsonRequest(w, r, &request); err != nil {
			return
		}
		mu.Lock()
		container := containers[request.UUID]
		var response skyhook.StatusResponse
		var err error
		if container == nil {
			err = fmt.Errorf("no container with that UUID")
		} else {
			for !container.Ready && container.Error == nil {
				cond.Wait()
			}
			if container.Error != nil {
				err = container.Error
			} else {
				response.Ready = true
				response.ExecBeginResponse = container.ExecBeginResponse
				response.BaseURL = container.BaseURL
			}
		}
		mu.Unlock()
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		skyhook.JsonResponse(w, response)
	})

	http.HandleFunc("/container/end", func(w http.ResponseWriter, r *http.Request) {
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
		container := containers[uuid]
		delete(containers, uuid)
		mu.Unlock()

		if container == nil {
			return
		}

		if mode == "docker" {
			err := exec.Command("docker", "rm", "--force", uuid).Run()
			if err != nil {
				panic(err)
			}
			container.Cmd.Wait()
		} else if mode == "process" {
			skyhook.JsonPost(container.BaseURL, "/exit", nil, nil)
			if err := container.Cmd.Wait(); err != nil {
				panic(err)
			}
		}

		log.Printf("[machine] container %s stopped", uuid)
	})

	log.Printf("starting on :%d", myPort)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", myPort), nil); err != nil {
		panic(err)
	}
}

func readContainerOutput(uuid string, stdout *bufio.Reader, stderr *bufio.Reader, jobID *int, coordinatorURL string) {
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
