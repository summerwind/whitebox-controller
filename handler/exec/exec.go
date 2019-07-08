package exec

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/summerwind/whitebox-controller/config"
	"github.com/summerwind/whitebox-controller/reconciler/state"
	"github.com/summerwind/whitebox-controller/webhook/injection"
)

type ExecHandler struct {
	command    string
	args       []string
	env        []string
	workingDir string
	timeout    time.Duration
	debug      bool
}

func New(c *config.ExecHandlerConfig) (*ExecHandler, error) {
	args := []string{}
	if c.Args != nil {
		args = append(args, c.Args...)
	}

	env := []string{}
	if c.Env != nil {
		for key, val := range c.Env {
			env = append(env, fmt.Sprintf("%s=%s", key, val))
		}
	}

	timeout := 60 * time.Second
	if c.Timeout != "" {
		var err error
		timeout, err = time.ParseDuration(c.Timeout)
		if err != nil {
			return nil, err
		}
	}

	return &ExecHandler{
		command:    c.Command,
		args:       args,
		env:        env,
		workingDir: c.WorkingDir,
		timeout:    timeout,
		debug:      c.Debug,
	}, nil
}

func (h *ExecHandler) HandleState(s *state.State) error {
	in, err := json.Marshal(s)
	if err != nil {
		return err
	}

	out, err := h.run(in)
	if err != nil {
		return err
	}

	err = json.Unmarshal(out, s)
	if err != nil {
		return err
	}

	return nil
}

func (h *ExecHandler) HandleAdmissionRequest(req admission.Request) (admission.Response, error) {
	res := admission.Response{}

	in, err := json.Marshal(&req)
	if err != nil {
		return res, err
	}

	out, err := h.run(in)
	if err != nil {
		return res, err
	}

	err = json.Unmarshal(out, &res)
	if err != nil {
		return res, err
	}

	return res, nil
}

func (h *ExecHandler) HandleInjectionRequest(req injection.Request) (injection.Response, error) {
	res := injection.Response{}

	in, err := json.Marshal(&req)
	if err != nil {
		return res, err
	}

	out, err := h.run(in)
	if err != nil {
		return res, err
	}

	err = json.Unmarshal(out, &res)
	if err != nil {
		return res, err
	}

	return res, nil
}

func (h *ExecHandler) run(buf []byte) ([]byte, error) {
	var stdout bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, h.command, h.args...)
	cmd.Stdin = bytes.NewReader(buf)
	cmd.Stdout = &stdout
	cmd.Env = append(os.Environ(), h.env...)
	cmd.Dir = h.workingDir

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	if h.debug {
		log("stdin", string(buf))

		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log("stderr", scanner.Text())
		}
	}

	err = cmd.Wait()
	if err != nil {
		return nil, err
	}

	if h.debug {
		log("stdout", stdout.String())
	}

	return stdout.Bytes(), nil
}

func log(stream, msg string) {
	fmt.Fprintf(os.Stderr, "[exec] %s: %s\n", stream, msg)
}
