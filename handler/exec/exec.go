package exec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/summerwind/whitebox-controller/config"
)

type ExecHandler struct {
	command    string
	args       []string
	env        []string
	workingDir string
	timeout    time.Duration
}

func NewHandler(hc *config.ExecHandlerConfig) (*ExecHandler, error) {
	args := []string{}
	if hc.Args != nil {
		args = append(args, hc.Args...)
	}

	env := []string{}
	if hc.Env != nil {
		for key, val := range hc.Env {
			env = append(env, fmt.Sprintf("%s=%s", key, val))
		}
	}

	timeout := 60 * time.Second
	if hc.Timeout != "" {
		var err error
		timeout, err = time.ParseDuration(hc.Timeout)
		if err != nil {
			return nil, err
		}
	}

	return &ExecHandler{
		command:    hc.Command,
		args:       args,
		env:        env,
		workingDir: hc.WorkingDir,
		timeout:    timeout,
	}, nil
}

func (h *ExecHandler) Run(buf []byte) ([]byte, error) {
	var stderr bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, h.command, h.args...)
	cmd.Stdin = bytes.NewReader(buf)
	cmd.Stderr = &stderr
	cmd.Env = append(os.Environ(), h.env...)
	cmd.Dir = h.workingDir

	out, err := cmd.Output()
	if err != nil {
		ee, ok := err.(*exec.ExitError)
		if ok {
			return nil, fmt.Errorf("%s: %s", err, string(ee.Stderr))
		}
		return nil, err
	}

	if len(out) == 0 {
		return nil, errors.New("empty command output")
	}

	return out, nil
}
