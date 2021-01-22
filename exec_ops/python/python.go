package python

import (
	"../../skyhook"
	"../../exec_ops"

	"fmt"
	"io"
	"sync"
)

// Data about one Apply call.
// Single goroutine reads stdout and passes information based on pendingKey structs.
type pendingKey struct {
	key string
	outputs map[string][]skyhook.Data
	builders map[string][]skyhook.ChunkBuilder
	cond *sync.Cond
	done bool
}

type JobPacket struct {
	Key string
	Type string
	Length int
}

type ResponsePacket struct {
	Type string
	Key string
	OutputKey string
	Length int
}

type PythonOp struct {
	url string
	node skyhook.ExecNode
	datasets []skyhook.Dataset

	cmd *skyhook.Cmd
	stdin io.WriteCloser
	stdout io.ReadCloser

	pending map[string]*pendingKey

	// error interacting with python process
	// after being set, this error is returned to any future Apply calls
	err error

	// lock on stdin
	writeLock sync.Mutex

	// lock on internal structures (pending, err, counter, etc.)
	mu sync.Mutex
}

func NewPythonOp(cmd *skyhook.Cmd, url string, node skyhook.ExecNode, outputDatasets []skyhook.Dataset) (*PythonOp, error) {
	stdin := cmd.Stdin()
	stdout := cmd.Stdout()

	// write meta packet
	var metaPacket struct {
		InputTypes []skyhook.DataType
		OutputTypes []skyhook.DataType
		Code string
	}
	metaPacket.Code = node.Params
	metaPacket.OutputTypes = node.DataTypes

	datasets, err := exec_ops.GetParentDatasets(url, node)
	if err != nil {
		return nil, fmt.Errorf("error getting parent datasets: %v", err)
	}
	for _, ds := range datasets {
		metaPacket.InputTypes = append(metaPacket.InputTypes, ds.DataType)
	}

	if err := skyhook.WriteJsonData(metaPacket, stdin); err != nil {
		return nil, err
	}

	op := &PythonOp{
		url: url,
		node: node,
		datasets: outputDatasets,
		cmd: cmd,
		stdin: stdin,
		stdout: stdout,
		pending: make(map[string]*pendingKey),
	}
	go op.readLoop()
	return op, nil
}

func (e *PythonOp) Parallelism() int {
	// python process is single-threaded, so there's no reason to run more than one task at a time
	return 1
}

func (e *PythonOp) readLoop() {
	var err error

	for {
		var resp ResponsePacket
		err = skyhook.ReadJsonData(e.stdout, &resp)
		if err != nil {
			break
		}

		if resp.Type == "data_data" {
			// read the datas
			datas := make([]skyhook.Data, len(e.node.DataTypes))
			for i, dtype := range e.node.DataTypes {
				dtype = skyhook.DataImpls[dtype].ChunkType
				datas[i], err = skyhook.DataImpls[dtype].DecodeStream(e.stdout)
				if err != nil {
					break
				}
			}
			if err != nil {
				break
			}

			// append the datas to the existing ones for this output key
			e.mu.Lock()
			pk := e.pending[resp.Key]
			if pk.builders[resp.OutputKey] == nil {
				pk.builders[resp.OutputKey] = make([]skyhook.ChunkBuilder, len(e.node.DataTypes))
				for i, dtype := range e.node.DataTypes {
					pk.builders[resp.OutputKey][i] = skyhook.DataImpls[dtype].Builder()
				}
			}
			for i, builder := range pk.builders[resp.OutputKey] {
				err = builder.Write(datas[i])
				if err != nil {
					break
				}
			}
			e.mu.Unlock()
			if err != nil {
				break
			}
		} else if resp.Type == "data_finish" {
			e.mu.Lock()
			pk := e.pending[resp.Key]
			pk.outputs[resp.OutputKey] = make([]skyhook.Data, len(e.node.DataTypes))
			for i, builder := range pk.builders[resp.OutputKey] {
				pk.outputs[resp.OutputKey][i], err = builder.Close()
				if err != nil {
					break
				}
			}
			e.mu.Unlock()
			if err != nil {
				break
			}
		} else if resp.Type == "finish" {
			e.mu.Lock()
			pk := e.pending[resp.Key]
			pk.done = true
			pk.cond.Broadcast()
			e.mu.Unlock()
		}
	}

	e.mu.Lock()
	if e.err == nil {
		e.err = err
	}
	for _, pk := range e.pending {
		pk.cond.Broadcast()
	}
	e.stdout.Close()
	e.stdin.Close()
	e.mu.Unlock()

}

func (e *PythonOp) Apply(task skyhook.ExecTask) error {
	// add pendingKey (and check if already err)
	e.mu.Lock()
	if e.err != nil {
		e.mu.Unlock()
		return e.err
	}

	pk := &pendingKey{
		key: task.Key,
		outputs: make(map[string][]skyhook.Data),
		builders: make(map[string][]skyhook.ChunkBuilder),
		cond: sync.NewCond(&e.mu),
	}
	e.pending[task.Key] = pk
	e.mu.Unlock()

	// write init packet
	e.writeLock.Lock()
	err := skyhook.WriteJsonData(JobPacket{
		Key: task.Key,
		Type: "init",
	}, e.stdin)
	e.writeLock.Unlock()
	if err != nil {
		return err
	}

	inputDatas := make([]skyhook.Data, len(task.Items))
	for i, input := range task.Items {
		data, err := input.LoadData()
		if err != nil {
			return err
		}
		inputDatas[i] = data
	}

	err = skyhook.SynchronizedReader(inputDatas, 32, func(pos int, length int, datas []skyhook.Data) error {
		e.writeLock.Lock()

		skyhook.WriteJsonData(JobPacket{
			Key: task.Key,
			Type: "job",
			Length: length,
		}, e.stdin)

		// just check the err on last write
		var err error
		for _, data := range datas {
			err = data.EncodeStream(e.stdin)
		}

		e.writeLock.Unlock()

		return err
	})

	// write finish packet
	// check err from SynchronizedReader after this packet is written
	e.writeLock.Lock()
	skyhook.WriteJsonData(JobPacket{
		Key: task.Key,
		Type: "finish",
	}, e.stdin)
	e.writeLock.Unlock()

	e.mu.Lock()
	// first check e.err because that may have caused the EncodeStream error
	if e.err != nil {
		e.mu.Unlock()
		return e.err
	} else if err != nil {
		e.mu.Unlock()
		return err
	}

	for !pk.done && e.err == nil {
		pk.cond.Wait()
	}
	e.mu.Unlock()

	if e.err != nil {
		return e.err
	}

	// write the outputs that were collected by readLoop
	for key, datas := range pk.outputs {
		for i := range datas {
			err := exec_ops.WriteItem(e.url, e.datasets[i], key, datas[i])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *PythonOp) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stdin.Close()
	e.stdout.Close()
	if e.cmd != nil {
		e.cmd.Wait()
		e.cmd = nil
		e.err = fmt.Errorf("closed")
	}
}

func init() {
	skyhook.ExecOpImpls["python"] = skyhook.ExecOpImpl{
		Requirements: func(url string, node skyhook.ExecNode) map[string]int {
			return nil
		},
		GetTasks: exec_ops.SimpleTasks,
		Prepare: func(url string, node skyhook.ExecNode, outputDatasets []skyhook.Dataset) (skyhook.ExecOp, error) {
			cmd := skyhook.Command("pynode-"+node.Name, skyhook.CommandOptions{}, "python3", "exec_ops/python/run.py")
			fmt.Println("python output datasets", outputDatasets)
			op, err := NewPythonOp(cmd, url, node, outputDatasets)
			if err != nil {
				return nil, err
			}
			return op, nil
		},
		Incremental: true,
		GetOutputKeys: exec_ops.MapGetOutputKeys,
		GetNeededInputs: exec_ops.MapGetNeededInputs,
		ImageName: func(url string, node skyhook.ExecNode) (string, error) {
			return "skyhookml/basic", nil
		},
	}
}
