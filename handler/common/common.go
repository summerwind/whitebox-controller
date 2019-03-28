package common

import (
	"errors"

	"github.com/summerwind/whitebox-controller/config"
	"github.com/summerwind/whitebox-controller/handler"
	"github.com/summerwind/whitebox-controller/handler/exec"
	"github.com/summerwind/whitebox-controller/handler/http"
)

func NewHandler(c *config.HandlerConfig) (handler.Handler, error) {
	var (
		h   handler.Handler
		err error
	)

	if c.Exec != nil {
		h, err = exec.NewHandler(c.Exec)
		if err != nil {
			return nil, err
		}
	}

	if c.HTTP != nil {
		h, err = http.NewHandler(c.HTTP)
		if err != nil {
			return nil, err
		}
	}

	if h == nil {
		return nil, errors.New("no handler found")
	}

	return h, nil
}
