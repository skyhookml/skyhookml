package yolov3

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Params struct {
	InputSize [2]int
	ConfigPath string
}

func (p Params) GetConfigPath() string {
	if p.ConfigPath == "" {
		return "cfg/yolov3.cfg"
	} else {
		return p.ConfigPath
	}
}

func CreateParams(fname string, p Params, training bool) {
	// prepare configuration with this width/height
	configPath := p.GetConfigPath()
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join("lib/darknet/", configPath)
	}
	bytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		panic(err)
	}
	file, err := os.Create(fname)
	if err != nil {
		panic(err)
	}
	for _, line := range strings.Split(string(bytes), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "width=") && p.InputSize[0] > 0 {
			line = fmt.Sprintf("width=%d", p.InputSize[0])
		} else if strings.HasPrefix(line, "height=") && p.InputSize[1] > 0 {
			line = fmt.Sprintf("height=%d", p.InputSize[1])
		} else if training && strings.HasPrefix(line, "batch=") {
			line = "batch=64"
		} else if training && strings.HasPrefix(line, "subdivisions=") {
			line = "subdivisions=8"
		}
		file.Write([]byte(line+"\n"))
	}
	file.Close()
}
