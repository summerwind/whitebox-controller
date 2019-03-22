package exec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/summerwind/whitebox-controller/config"
	"github.com/summerwind/whitebox-controller/handler"
)

type ExecHandler struct {
	config *config.ExecHandlerConfig
	env    []string
}

func NewHandler(hc *config.ExecHandlerConfig) *ExecHandler {
	env := []string{}
	if hc.Env != nil {
		for key, val := range hc.Env {
			env = append(env, fmt.Sprintf("%s=%s", key, val))
		}
	}
	return &ExecHandler{
		config: hc,
		env:    env,
	}
}

func (h *ExecHandler) Run(req *handler.Request) (*handler.Response, error) {
	buf, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	if h.config.Args == nil {
		h.config.Args = []string{}
	}

	cmd := exec.Command(h.config.Command, h.config.Args...)
	cmd.Env = append(os.Environ(), h.env...)
	cmd.Stdin = bytes.NewReader(buf)

	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	res := &handler.Response{}
	err = json.Unmarshal(out, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}
