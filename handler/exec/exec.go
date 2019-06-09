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

	"github.com/summerwind/whitebox-controller/config"
	"github.com/summerwind/whitebox-controller/reconciler/state"
	"github.com/summerwind/whitebox-controller/webhook/injection"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type ExecHandler struct {
	command    string
	args       []string
	env        []string
	workingDir string
	timeout    time.Duration
	debug      bool
}

func NewHandler(c *config.ExecHandlerConfig) (*ExecHandler, error) {
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

func (h *ExecHandler) Reconcile(s *state.State) (*state.State, error) {
	buf, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	out, err := h.run(buf)
	if err != nil {
		return nil, err
	}

	newState := state.State{}
	err = json.Unmarshal(out, &newState)
	if err != nil {
		return nil, err
	}

	return &newState, nil
}

func (h *ExecHandler) Finalize(s *state.State) (*state.State, error) {
	return h.Reconcile(s)
}

func (h *ExecHandler) Validate(req *admission.Request) (*admission.Response, error) {
	buf, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	out, err := h.run(buf)
	if err != nil {
		return nil, err
	}

	res := &admission.Response{}
	err = json.Unmarshal(out, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (h *ExecHandler) Mutate(req *admission.Request) (*admission.Response, error) {
	buf, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	out, err := h.run(buf)
	if err != nil {
		return nil, err
	}

	res := &admission.Response{}
	err = json.Unmarshal(out, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (h *ExecHandler) Inject(req *injection.Request) (*injection.Response, error) {
	buf, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	out, err := h.run(buf)
	if err != nil {
		return nil, err
	}

	res := &injection.Response{}
	err = json.Unmarshal(out, res)
	if err != nil {
		return nil, err
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
