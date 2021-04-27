package python

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
)

type Params struct {
	Code string
	Outputs []skyhook.ExecOutput
}

type Packet struct {
	RequestID int
	Name string
	JSON string
}

type PythonOp struct {
	url string
	outputDatasets []skyhook.Dataset

	cmd *skyhook.Cmd
	stdin io.WriteCloser
	stdout io.ReadCloser
	httpServer HttpServer

	// Next request ID to use, starting from 0.
	// Request IDs differentiate ExecOp function calls that we encapsulate on stdin.
	// This way we can receive responses on stdout.
	nextRequestID int
	// Requests waiting for a response on stdout.
	// A response is just a JSON-encoded string.
	pendingRequests map[int]*string

	// error interacting with python process
	// after being set, this error is returned to any future Apply calls
	err error

	// lock on internal structures (pending, err, counter, etc.) and on stdin
	mu sync.Mutex
	// Condition variable used by pending requests to wait for pendingRequests[id] != nil.
	// (or err != nil)
	cond *sync.Cond
}

func NewPythonOp(cmd *skyhook.Cmd, url string, params Params, inputDatasets []skyhook.Dataset, outputDatasets []skyhook.Dataset) (*PythonOp, error) {
	stdin := cmd.Stdin()
	stdout := cmd.Stdout()

	// Initialize local HTTP server.
	httpServer, err := NewHttpServer(url)
	if err != nil {
		return nil, err
	}

	// Write metadata packet, which enables Python script to initialize the custom Python operator.
	var metaPacket struct {
		Inputs []skyhook.Dataset
		Outputs []skyhook.Dataset
		Code string
		URL string
		Port int
	}
	metaPacket.Inputs = inputDatasets
	metaPacket.Outputs = outputDatasets
	metaPacket.Code = params.Code
	metaPacket.URL = url
	metaPacket.Port = httpServer.Port

	if err := skyhook.WriteJsonData(metaPacket, stdin); err != nil {
		return nil, err
	}

	op := &PythonOp{
		url: url,
		outputDatasets: outputDatasets,
		cmd: cmd,
		stdin: stdin,
		stdout: stdout,
		httpServer: httpServer,
		pendingRequests: make(map[int]*string),
	}
	op.cond = sync.NewCond(&op.mu)
	go op.readLoop()
	return op, nil
}

// Wait for the response to a request.
// On success, decodes the response into x.
// Returns error if e.err is set or if decoding fails.
// Caller must have the lock (e.mu).
func (e *PythonOp) waitOnRequest(requestID int, x interface{}) error {
	for e.pendingRequests[requestID] == nil && e.err == nil {
		e.cond.Wait()
	}
	if e.err != nil {
		return e.err
	}
	if x != nil {
		err := json.Unmarshal([]byte(*e.pendingRequests[requestID]), x)
		if err != nil {
			return err
		}
	}
	return nil
}

// Prepare a new packet with a fresh request ID.
// Caller must have the lock.
func (e *PythonOp) makePacket(name string, x interface{}) Packet {
	packet := Packet{
		RequestID: e.nextRequestID,
		Name: name,
		JSON: string(skyhook.JsonMarshal(x)),
	}
	e.nextRequestID++
	return packet
}

// Make a request with the given request/response objects.
func (e *PythonOp) request(name string, req interface{}, resp interface{}) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	packet := e.makePacket(name, req)

	// We ignore the error here since it's generally not a useful message.
	// If writing fails, it means readLoop probably has terminated and that will
	// eventually populate e.err with a more useful error (which we get from
	// waitOnRequest).
	skyhook.WriteJsonData(packet, e.stdin)

	err := e.waitOnRequest(packet.RequestID, resp)
	if err != nil {
		return err
	}
	return nil
}

func (e *PythonOp) Parallelism() int {
	var response int
	err := e.request("parallelism", nil, &response)
	if err != nil {
		return 1
	}
	return response
}

// Close the e.cmd if set.
// Caller must have the lock.
func (e *PythonOp) closeCmd() {
	e.stdout.Close()
	e.stdin.Close()
	if e.cmd != nil {
		err := e.cmd.Wait()
		e.cmd = nil
		if err != nil && e.err == nil {
			e.err = err
		}
	}
}

func (e *PythonOp) readLoop() {
	signature := "skjson"
	var err error
	rd := bufio.NewReader(e.stdout)
	for {
		var line string
		line, err = rd.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, signature) {
			fmt.Println(line)
			continue
		}

		line = line[len(signature):]
		var packet Packet
		skyhook.JsonUnmarshal([]byte(line), &packet)

		e.mu.Lock()
		e.pendingRequests[packet.RequestID] = &packet.JSON
		e.cond.Broadcast()
		e.mu.Unlock()
	}

	e.mu.Lock()
	// We prioritize setting e.err to the error returned by cmd.Wait,
	// since that error has more information (i.e., contains the stderr output).
	e.closeCmd()
	if e.err == nil {
		e.err = err
	}
	e.cond.Broadcast()
	e.mu.Unlock()
}

func (e *PythonOp) Apply(task skyhook.ExecTask) error {
	err := e.request("apply", task, nil)
	if err != nil {
		return err
	}
	return nil
}

func (e *PythonOp) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.httpServer.Close()
	e.closeCmd()
	if e.err != nil {
		e.err = fmt.Errorf("closed")
	}
}

func init() {
	getOp := func(url string, node skyhook.Runnable) (*PythonOp, error) {
		var params Params
		if err := exec_ops.DecodeParams(node, &params, false); err != nil {
			return nil, err
		}
		var flatOutputs []skyhook.Dataset
		for _, output := range params.Outputs {
			flatOutputs = append(flatOutputs, node.OutputDatasets[output.Name])
		}

		cmd := skyhook.Command(
			"pynode-"+node.Name,
			skyhook.CommandOptions{AllStderrLines: true},
			"python3", "exec_ops/pythonv2/run.py",
		)
		return NewPythonOp(cmd, url, params, node.InputDatasets["inputs"], flatOutputs)
	}

	skyhook.AddExecOpImpl(skyhook.ExecOpImpl{
		Config: skyhook.ExecOpConfig{
			ID: "pythonv2",
			Name: "Python",
			Description: "Express a Python function for the system to execute",
		},
		Inputs: []skyhook.ExecInput{{Name: "inputs", Variable: true}},
		GetOutputs: func(rawParams string, inputTypes map[string][]skyhook.DataType) []skyhook.ExecOutput {
			// outputs are specified by user in Params
			var params Params
			err := json.Unmarshal([]byte(rawParams), &params)
			if err != nil {
				return []skyhook.ExecOutput{}
			}
			return params.Outputs
		},
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: func(node skyhook.Runnable, rawItems map[string][][]skyhook.Item) ([]skyhook.ExecTask, error) {
			op, err := getOp("", node)
			if err != nil {
				return nil, err
			}
			defer op.Close()
			var tasks []skyhook.ExecTask
			err = op.request("get_tasks", rawItems, &tasks)
			if err != nil {
				return nil, err
			}
			return tasks, nil
		},
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			return getOp(url, node)
		},
		Incremental: true,
		GetOutputKeys: exec_ops.MapGetOutputKeys,
		GetNeededInputs: exec_ops.MapGetNeededInputs,
		ImageName: "skyhookml/basic",
	})
}
