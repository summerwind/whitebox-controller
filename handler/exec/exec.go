package exec

import (
	"bufio"
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

func (h *ExecHandler) Run(buf []byte) ([]byte, error) {
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
		fmt.Printf("[Exec] stdin: %s\n", string(buf))

		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Printf("[Exec] stderr: %s\n", scanner.Text())
		}
	}

	err = cmd.Wait()
	if err != nil {
		return nil, err
	}

	if h.debug {
		fmt.Printf("[Exec] stdout: %s\n", stdout.String())
	}

	if len(stdout.Bytes()) == 0 {
		return nil, errors.New("no output of command")
	}

	return stdout.Bytes(), nil
}
