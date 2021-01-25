package main

import (
	"./skyhook"
	gouuid "github.com/google/uuid"

	_ "./ops"

	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("usage: ./worker [external IP] [port] [skyhook URL]")
		fmt.Println("example: ./worker localhost 8081 http://localhost:8080")
		return
	}
	myIP := os.Args[1]
	myPort := skyhook.ParseInt(os.Args[2])
	coordinatorURL := os.Args[3]

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

	http.HandleFunc("/exec/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(404)
			return
		}

		var request skyhook.ExecBeginRequest
		if err := skyhook.ParseJsonRequest(w, r, &request); err != nil {
			return
		}

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

		opImpl := skyhook.GetExecOpImpl(request.Node.Op)
		imageName, err := opImpl.ImageName(coordinatorURL, request.Node)
		if err != nil {
			panic(err)
		}

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
				"go", "run", "container.go", fmt.Sprintf(":%d", containerPort),
			)
		}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			panic(err)
		}
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			panic(err)
		}
		log.Printf("[machine] container %s started", uuid)

		mu.Lock()
		containers[uuid].Cmd = cmd
		mu.Unlock()

		// once container is ready, we need to forward this /exec/start to the container
		rd := bufio.NewReader(stdout)
		_, err = rd.ReadString('\n')
		if err != nil {
			panic(err)
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

	http.HandleFunc("/train/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(404)
			return
		}

		var request skyhook.TrainBeginRequest
		if err := skyhook.ParseJsonRequest(w, r, &request); err != nil {
			return
		}

		uuid := gouuid.New().String()

		mu.Lock()
		containerPort := getPort()
		containers[uuid] = &Cmd{
			Cmd: nil,
			Port: containerPort,
		}
		mu.Unlock()

		op := skyhook.GetTrainOp(request.Node.Op)
		imageName, err := op.ImageName(coordinatorURL, request.Node)
		if err != nil {
			panic(err)
		}
		cmd := exec.Command(
			"docker", "run",
			"--mount", fmt.Sprintf("\"src=%s\",target=/usr/src/app/skyhook/items,type=bind", filepath.Join(workingDir, "items")),
			"--mount", fmt.Sprintf("\"src=%s\",target=/usr/src/app/skyhook/models,type=bind", filepath.Join(workingDir, "models")),
			"--gpus", "all",
			"-p", fmt.Sprintf("%d:8080", containerPort),
			"--name", uuid,
			imageName,
		)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			panic(err)
		}
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			panic(err)
		}
		log.Printf("[machine] container %s started", uuid)

		mu.Lock()
		containers[uuid].Cmd = cmd
		mu.Unlock()

		// once container is ready, we need to forward this /train/start to the container
		containerBaseURL := fmt.Sprintf("http://localhost:%d", containerPort)
		rd := bufio.NewReader(stdout)
		_, err = rd.ReadString('\n')
		if err != nil {
			panic(err)
		}
		go func() {
			for {
				line, err := rd.ReadString('\n')
				if err != nil {
					break
				}
				log.Printf("[container %s] %s", uuid, line)
			}
		}()

		request.CoordinatorURL = coordinatorURL
		var response skyhook.TrainBeginResponse
		err = skyhook.JsonPost(containerBaseURL, "/train/start", request, &response)
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
