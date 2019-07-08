package common

import (
	"errors"
	"os"

	"github.com/summerwind/whitebox-controller/config"
	"github.com/summerwind/whitebox-controller/handler"
	"github.com/summerwind/whitebox-controller/handler/exec"
	"github.com/summerwind/whitebox-controller/handler/http"
)

// The name of environment variable to enable debug log.
const debugEnvVar = "WHITEBOX_DEBUG"

var errNoHandler = errors.New("no handler found")

// NewStateHandler returns StateHandler based on specified HandlerConfig.
func NewStateHandler(c *config.HandlerConfig) (handler.StateHandler, error) {
	var debug bool

	if c.StateHandler != nil {
		return c.StateHandler, nil
	}

	if os.Getenv(debugEnvVar) != "" {
		debug = true
	}

	if c.Exec != nil {
		c.Exec.Debug = (c.Exec.Debug || debug)
		return exec.New(c.Exec)
	}

	if c.HTTP != nil {
		c.HTTP.Debug = (c.HTTP.Debug || debug)
		return http.New(c.HTTP)
	}

	return nil, errNoHandler
}

// NewAdmissionRequestHandler returns AdmissionRequestHandler based on specified HandlerConfig.
func NewAdmissionRequestHandler(c *config.HandlerConfig) (handler.AdmissionRequestHandler, error) {
	var debug bool

	if c.AdmissionRequestHandler != nil {
		return c.AdmissionRequestHandler, nil
	}

	if os.Getenv(debugEnvVar) != "" {
		debug = true
	}

	if c.Exec != nil {
		c.Exec.Debug = (c.Exec.Debug || debug)
		return exec.New(c.Exec)
	}

	if c.HTTP != nil {
		c.HTTP.Debug = (c.HTTP.Debug || debug)
		return http.New(c.HTTP)
	}

	return nil, errNoHandler
}

// NewInjectionRequestHandler returns InjectionRequestHandler based on specified HandlerConfig.
func NewInjectionRequestHandler(c *config.HandlerConfig) (handler.InjectionRequestHandler, error) {
	var debug bool

	if c.InjectionRequestHandler != nil {
		return c.InjectionRequestHandler, nil
	}

	if os.Getenv(debugEnvVar) != "" {
		debug = true
	}

	if c.Exec != nil {
		c.Exec.Debug = (c.Exec.Debug || debug)
		return exec.New(c.Exec)
	}

	if c.HTTP != nil {
		c.HTTP.Debug = (c.HTTP.Debug || debug)
		return http.New(c.HTTP)
	}

	return nil, errNoHandler
}
