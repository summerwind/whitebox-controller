package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type HandlerType string

const (
	HandlerTypeExec HandlerType = "exec"
)

type Config struct {
	Resource           schema.GroupVersionKind   `json:"resource"`
	DependentResources []schema.GroupVersionKind `json:"dependentResources"`
	Handlers           map[string]HandlerConfig  `json:"handlers"`
}

func LoadFile(p string) (*Config, error) {
	buf, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}

	c := &Config{
		DependentResources: []schema.GroupVersionKind{},
		Handlers:           map[string]HandlerConfig{},
	}

	err = yaml.Unmarshal(buf, c)
	if err != nil {
		return nil, err
	}

	err = c.Validate()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Config) Validate() error {
	if c.Resource.Empty() {
		return errors.New("resource is empty")
	}

	for i, dep := range c.DependentResources {
		if dep.Empty() {
			return fmt.Errorf("dependentResource[%d] is empty", i)
		}
	}

	for key, h := range c.Handlers {
		err := h.Validate()
		if err != nil {
			return fmt.Errorf("handlers[%s]: %v", key, err)
		}
	}

	return nil
}

type HandlerConfig struct {
	Exec *ExecHandlerConfig `json:"exec"`
}

func (hc *HandlerConfig) Validate() error {
	if hc.Exec == nil {
		return errors.New("exec must be specified")
	}

	err := hc.Exec.Validate()
	if err != nil {
		return err
	}

	return nil
}

type ExecHandlerConfig struct {
	Command    string            `json:"command"`
	Args       []string          `json:"args"`
	WorkingDir string            `json:"workingDir"`
	Env        map[string]string `json:"env"`
	Timeout    string            `json:"timeout"`
}

func (ehc ExecHandlerConfig) Validate() error {
	if ehc.Command == "" {
		return errors.New("command must be specified")
	}

	if ehc.Timeout != "" {
		_, err := time.ParseDuration(ehc.Timeout)
		if err != nil {
			return fmt.Errorf("timeout is not valid: %v", err)
		}
	}

	return nil
}
