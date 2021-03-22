package app

import (
	"github.com/skyhookml/skyhookml/skyhook"

	"log"
	"net/http"
	"sync"
)

var Workers = make(map[string]bool)
var workersMu sync.Mutex

func AcquireWorker() string {
	workersMu.Lock()
	defer workersMu.Unlock()
	for workerURL, ok := range Workers {
		if !ok {
			continue
		}
		Workers[workerURL] = false
		return workerURL
	}
	return ""
}

func ReleaseWorker(url string) {
	workersMu.Lock()
	Workers[url] = true
	workersMu.Unlock()
}

func init() {
	Router.HandleFunc("/worker/init", func(w http.ResponseWriter, r *http.Request) {
		var request skyhook.WorkerInitRequest
		if err := skyhook.ParseJsonRequest(w, r, &request); err != nil {
			return
		}
		log.Printf("[worker] add new worker at %s", request.BaseURL)
		workersMu.Lock()
		Workers[request.BaseURL] = true
		workersMu.Unlock()
	})
}
