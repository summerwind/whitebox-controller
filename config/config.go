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
	Controllers []ControllerConfig `json:"controllers"`
}

func LoadFile(p string) (*Config, error) {
	buf, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}

	c := &Config{}
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
	for i, controller := range c.Controllers {
		err := controller.Validate()
		if err != nil {
			return fmt.Errorf("controller[%d]: %v", i, err)
		}
	}

	return nil
}

type ControllerConfig struct {
	Name               string
	Resource           schema.GroupVersionKind   `json:"resource"`
	DependentResources []schema.GroupVersionKind `json:"dependentResources"`
	Reconciler         HandlerConfig             `json:"reconciler"`
	Syncer             SyncerConfig              `json:"syncer"`
}

func (cc *ControllerConfig) Validate() error {
	if cc.Name == "" {
		return errors.New("name must be specified")
	}

	if cc.Resource.Empty() {
		return errors.New("resource is empty")
	}

	for i, dep := range cc.DependentResources {
		if dep.Empty() {
			return fmt.Errorf("dependentResource[%d] is empty", i)
		}
	}

	err := cc.Reconciler.Validate()
	if err != nil {
		return fmt.Errorf("reconciler: %v", err)
	}

	err = cc.Syncer.Validate()
	if err != nil {
		return fmt.Errorf("syncer: %v", err)
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
			return fmt.Errorf("invalid timeout: %v", err)
		}
	}

	return nil
}

type SyncerConfig struct {
	Interval string `json:"interval"`
}

func (sc SyncerConfig) Validate() error {
	if sc.Interval != "" {
		_, err := time.ParseDuration(sc.Interval)
		if err != nil {
			return fmt.Errorf("invalid interval: %v", err)
		}
	}

	return nil
}
