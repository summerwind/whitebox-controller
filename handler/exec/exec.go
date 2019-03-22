package exec

import (
	"bytes"
	"encoding/json"
	"os/exec"

	"github.com/summerwind/whitebox-controller/config"
	"github.com/summerwind/whitebox-controller/handler"
)

type ExecHandler struct {
	config *config.ExecHandlerConfig
}

func NewHandler(hc *config.ExecHandlerConfig) *ExecHandler {
	return &ExecHandler{config: hc}
}

func (h *ExecHandler) Run(req *handler.Request) (*handler.Response, error) {
	res := &handler.Response{}

	buf, err := json.Marshal(req)
	if err != nil {
		return res, err
	}

	if h.config.Args == nil {
		h.config.Args = []string{}
	}

	cmd := exec.Command(h.config.Command, h.config.Args...)
	cmd.Stdin = bytes.NewReader(buf)

	out, err := cmd.Output()
	if err != nil {
		return res, err
	}

	err = json.Unmarshal(out, res)
	if err != nil {
		return res, err
	}

	return res, nil
}
