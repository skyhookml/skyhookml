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
	startContainer := func(uuid string, imageName string, jobID *int, coordinatorURL string, instanceID string, needsGPU bool) (string, error) {
		mu.Lock()
		containerPort := getPort()
		containerBaseURL := fmt.Sprintf("http://%s:%d", myIP, containerPort)
		containers[uuid].Port = containerPort
		containers[uuid].BaseURL = containerBaseURL
		mu.Unlock()

		setError := func(err error) {
			mu.Lock()
			containers[uuid].Error = err
			cond.Broadcast()
			mu.Unlock()
		}

		var cmd *exec.Cmd
		if mode == "docker" {
			dataDir := filepath.Join(workingDir, "data")
			if instanceID != "" {
				dataDir = filepath.Join(dataDir, filepath.Base(instanceID))
			}
			args := []string{
				"docker", "run",
				"--mount", fmt.Sprintf("\"src=%s\",target=/usr/src/app/skyhook/data,type=bind", dataDir),
				"-p", fmt.Sprintf("%d:8080", containerPort),
				"--name", uuid,
				// pytorch DataLoader needs more than tiny default 64MB shared memory
				"--shm-size", "1G",
			}
			if needsGPU {
				args = append(args, []string{
					"--gpus", "all",
				}...)
			}
			args = append(args, []string{
				strings.ReplaceAll(imageName, "skyhookml/", "skyhookml/demo-"),
			}...)
			cmd = exec.Command(args[0], args[1:]...)
		} else if mode == "process" {
			cmd = exec.Command(
				"go", "run", "cmd/container.go", fmt.Sprintf(":%d", containerPort),
			)
		}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			setError(err)
			return "", err
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			setError(err)
			return "", err
		}
		if err := cmd.Start(); err != nil {
			setError(err)
			return "", err
		}
		log.Printf("[worker] container %s started", uuid)

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

	stopContainer := func(uuid string, container *Container) {
		if mode == "docker" {
			err := exec.Command("docker", "rm", "--force", uuid).Run()
			if err != nil {
				log.Printf("error stopping docker container: %v", err)
			}
			container.Cmd.Wait()
		} else if mode == "process" {
			skyhook.JsonPost(container.BaseURL, "/exit", nil, nil)
			if err := container.Cmd.Wait(); err != nil {
				log.Printf("error waiting for process to exit: %v", err)
			}
		}
		log.Printf("[worker] container %s stopped", uuid)
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
			needsGPU := request.Node.GetOp().Requirements(request.Node)["gpu"] > 0

			baseURL, err := startContainer(uuid, imageName, request.JobID, request.CoordinatorURL, request.InstanceID, needsGPU)
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
				mu.Lock()
				container := containers[uuid]
				container.Error = err
				cond.Broadcast()
				mu.Unlock()
				stopContainer(uuid, container)
				return
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

		stopContainer(uuid, container)
	})

	log.Printf("starting on :%d", myPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", myPort), nil))
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
					log.Printf("error posting job update: %v", err)
					continue
				}
			}
		}()
	}
}
