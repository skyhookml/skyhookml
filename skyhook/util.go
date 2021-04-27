package skyhook

import (
	crypto_rand "crypto/rand"
	"bytes"
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	urllib "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rubenfonseca/fastimage"
)

func ReadTextFile(fname string) string {
	bytes, err := ioutil.ReadFile(fname)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

func ReadJSONFile(fname string, res interface{}) {
	bytes, err := ioutil.ReadFile(fname)
	if err != nil {
		panic(err)
	}
	JsonUnmarshal(bytes, res)
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func JsonMarshal(x interface{}) []byte {
	bytes, err := json.Marshal(x)
	if err != nil {
		panic(err)
	}
	return bytes
}

func JsonUnmarshal(bytes []byte, x interface{}) {
	err := json.Unmarshal(bytes, x)
	if err != nil {
		panic(err)
	}
}

func JsonResponse(w http.ResponseWriter, x interface{}) {
	bytes := JsonMarshal(x)
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}

func ParseJsonRequest(w http.ResponseWriter, r *http.Request, x interface{}) error {
	bytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("json decode error: %v", err), 400)
		return err
	}
	if err := json.Unmarshal(bytes, x); err != nil {
		http.Error(w, fmt.Sprintf("json decode error: %v", err), 400)
		return err
	}
	return nil
}

func ParseJsonResponse(resp *http.Response, response interface{}) error {
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error performing HTTP request: %v", err)
	} else if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(bytes))
	}
	if response != nil {
		JsonUnmarshal(bytes, response)
	}
	return nil
}

func JsonGet(baseURL string, path string, response interface{}) error {
	resp, err := http.Get(baseURL + path)
	if err != nil {
		return fmt.Errorf("error performing HTTP request: %v", err)
	}
	err = ParseJsonResponse(resp, response)
	if err != nil {
		return fmt.Errorf("[GET %s] %v", baseURL+path, err)
	}
	return nil
}

func JsonPost(baseURL string, path string, request interface{}, response interface{}) error {
	var body io.Reader
	if request != nil {
		body = bytes.NewBuffer(JsonMarshal(request))
	}
	resp, err := http.Post(baseURL + path, "application/json", body)
	if err != nil {
		return fmt.Errorf("error performing HTTP request (%s): %v", baseURL+path, err)
	}
	err = ParseJsonResponse(resp, response)
	if err != nil {
		return fmt.Errorf("[POST %s] %v", baseURL+path, err)
	}
	return nil
}

func JsonPostForm(baseURL string, path string, request urllib.Values, response interface{}) error {
	resp, err := http.PostForm(baseURL+path, request)
	if err != nil {
		return fmt.Errorf("error performing HTTP request (%s): %v", baseURL+path, err)
	}
	err = ParseJsonResponse(resp, response)
	if err != nil {
		return fmt.Errorf("[POST %s] %v", baseURL+path, err)
	}
	return nil
}

func ParseInt(str string) int {
	x, err := strconv.Atoi(str)
	if err != nil {
		panic(err)
	}
	return x
}

func ParseFloat(str string) float64 {
	x, err := strconv.ParseFloat(str, 64)
	if err != nil {
		panic(err)
	}
	return x
}

func CopyFile(src string, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}
	return nil
}

const Debug bool = false

type Cmd struct {
	prefix string
	cmd *exec.Cmd
	stdin io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
	// if not nil, means PrintStderr will send last line(s) it got before exiting
	stderrCh chan []string
	closed bool
}

func (cmd *Cmd) Stdin() io.WriteCloser {
	return cmd.stdin
}

func (cmd *Cmd) Stdout() io.ReadCloser {
	return cmd.stdout
}

func (cmd *Cmd) Stderr() io.ReadCloser {
	return cmd.stderr
}

type CmdError struct {
	ExitError error
	Lines []string
}

func (e CmdError) Error() string {
	var linesPart string
	if len(e.Lines) > 0 {
		linesPart = fmt.Sprintf(" (%s)", e.Lines[len(e.Lines)-1])
	}
	return fmt.Sprintf("exit error: %v", e.ExitError) + linesPart
}

func (cmd *Cmd) Wait() error {
	if cmd.closed {
		panic(fmt.Errorf("closed twice"))
	}
	cmd.closed = true
	if cmd.stdin != nil {
		cmd.stdin.Close()
	}
	if cmd.stdout != nil {
		cmd.stdout.Close()
	}
	var lastLines []string
	if cmd.stderrCh != nil {
		lastLines = <- cmd.stderrCh
	}
	err := cmd.cmd.Wait()
	if err != nil {
		myerr := CmdError{
			ExitError: err,
			Lines: lastLines,
		}
		log.Printf("[%s] %v", cmd.prefix, myerr.Error())
		return myerr
	}
	return nil
}

func (cmd *Cmd) printStderr(opts CommandOptions) {
	rd := bufio.NewReader(cmd.stderr)
	var lastLines []string
	for {
		line, err := rd.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		line = strings.TrimRight(line, "\n")
		if opts.AllStderrLines {
			lastLines = append(lastLines, line)
		} else {
			lastLines = []string{line}
		}
		if !opts.OnlyDebug || Debug {
			log.Printf("[%s] %s", cmd.prefix, line)
		}
	}
	cmd.stderrCh <- lastLines
}

type CommandOptions struct {
	NoStdin bool
	NoStdout bool
	NoStderr bool
	NoPrintStderr bool
	// Function to arbitrary modify the exec.Cmd, e.g., set working directory.
	// This is called just before starting the process.
	F func(*exec.Cmd)
	// Whether to only print stderr if debug mode is on.
	OnlyDebug bool
	// Whether to keep not just the last stderr line, but all lines, in case of error.
	AllStderrLines bool
}

func Command(prefix string, opts CommandOptions, command string, args ...string) *Cmd {
	log.Printf("[util] %s %v", command, args)
	cmd := exec.Command(command, args...)
	var stdin io.WriteCloser
	if !opts.NoStdin {
		var err error
		stdin, err = cmd.StdinPipe()
		if err != nil {
			panic(err)
		}
	}
	var stdout io.ReadCloser
	if !opts.NoStdout {
		var err error
		stdout, err = cmd.StdoutPipe()
		if err != nil {
			panic(err)
		}
	}
	var stderr io.ReadCloser
	if !opts.NoStderr {
		var err error
		stderr, err = cmd.StderrPipe()
		if err != nil {
			panic(err)
		}
	}
	if opts.F != nil {
		opts.F(cmd)
	}
	if err := cmd.Start(); err != nil {
		panic(err)
	}
	mycmd := &Cmd{
		prefix: prefix,
		cmd: cmd,
		stdin: stdin,
		stdout: stdout,
		stderr: stderr,
	}
	if stderr != nil && !opts.NoPrintStderr {
		mycmd.stderrCh = make(chan []string)
		go mycmd.printStderr(opts)
	}
	return mycmd
}

func Mod(a, b int) int {
	x := a%b
	if x < 0 {
		x = x+b
	}
	return x
}

func SeedRand() {
	var b [8]byte
	_, err := crypto_rand.Read(b[:])
	if err != nil {
		panic(err)
	}
	rand.Seed(int64(binary.BigEndian.Uint64(b[:])))
}

func Clip(x, lo, hi int) int {
	if x < lo {
		return lo
	} else if x > hi {
		return hi
	} else {
		return x
	}
}

func CopyOrSymlink(srcFname string, dstFname string, symlink bool) error {
	if symlink {
		// need to make sure srcFname is an absolute path
		// since it may not be rooted relative to the directory of dstFname
		var err error
		srcFname, err = filepath.Abs(srcFname)
		if err != nil {
			return err
		}
		return os.Symlink(srcFname, dstFname)
	} else {
		return CopyFile(srcFname, dstFname)
	}
}

func GetImageDimsFromFile(fname string) ([2]int, error) {
	var dims [2]int
	file, err := os.Open(fname)
	if err != nil {
		return dims, err
	}
	_, size, err := fastimage.DetectImageTypeFromReader(file)
	if err != nil {
		return dims, err
	} else if size == nil {
		return dims, fmt.Errorf("unknown image format")
	}
	dims = [2]int{int(size.Width), int(size.Height)}
	return dims, nil
}

// Like filepath.Ext but doesn't include the ".".
func Ext(fname string) string {
	ext := filepath.Ext(fname)
	if len(ext) == 0 || ext[0] != '.' {
		return ext
	} else {
		return ext[1:]
	}
}

func FileExists(fname string) bool {
	_, err := os.Stat(fname)
	return err == nil
}
