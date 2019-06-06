package common

import (
	"errors"
	"os"

	"github.com/summerwind/whitebox-controller/config"
	"github.com/summerwind/whitebox-controller/handler"
	"github.com/summerwind/whitebox-controller/handler/exec"
	"github.com/summerwind/whitebox-controller/handler/http"
)

const debugEnv = "WHITEBOX_DEBUG"

func NewHandler(c *config.HandlerConfig) (handler.Handler, error) {
	var (
		h     handler.Handler
		err   error
		debug bool
	)

	if os.Getenv(debugEnv) != "" {
		debug = true
	}

	if c.Exec != nil {
		c.Exec.Debug = (c.Exec.Debug || debug)
		h, err = exec.NewHandler(c.Exec)
		if err != nil {
			return nil, err
		}
	}

	if c.HTTP != nil {
		c.HTTP.Debug = (c.Exec.Debug || debug)
		h, err = http.NewHandler(c.HTTP)
		if err != nil {
			return nil, err
		}
	}

	if c.Func != nil {
		h = c.Func.Handler
	}

	if h == nil {
		return nil, errors.New("no handler found")
	}

	return h, nil
}
