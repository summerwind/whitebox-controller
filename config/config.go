package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/summerwind/whitebox-controller/handler"
)

type Config struct {
	Name      string            `json:"name,omitempty"`
	Resources []*ResourceConfig `json:"resources"`
	Webhook   *ServerConfig     `json:"webhook,omitempty"`
	Metrics   *ServerConfig     `json:"metrics,omitempty"`
}

func LoadFile(p string) (*Config, error) {
	buf, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %v", p, err)
	}

	c := &Config{}
	err = yaml.Unmarshal(buf, c)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %v", p, err)
	}

	return c, nil
}

func (c *Config) Validate() error {
	if len(c.Resources) == 0 {
		return errors.New("at least one resource must be specified")
	}

	for i, r := range c.Resources {
		err := r.Validate()
		if err != nil {
			return fmt.Errorf("resources[%d]: %v", i, err)
		}
	}

	if c.Webhook != nil {
		err := c.Webhook.Validate()
		if err != nil {
			return fmt.Errorf("webhook: %v", err)
		}
	}

	if c.Metrics != nil {
		err := c.Metrics.Validate()
		if err != nil {
			return fmt.Errorf("metrics: %v", err)
		}
	}

	return nil
}

type ResourceConfig struct {
	schema.GroupVersionKind

	Dependents []DependentConfig `json:"dependents,omitempty"`
	References []ReferenceConfig `json:"references,omitempty"`

	Reconciler   *ReconcilerConfig `json:"reconciler,omitempty"`
	Finalizer    *HandlerConfig    `json:"finalizer,omitempty"`
	ResyncPeriod string            `json:"resyncPeriod,omitempty"`

	Validator *HandlerConfig  `json:"validator,omitempty"`
	Mutator   *HandlerConfig  `json:"mutator,omitempty"`
	Injector  *InjectorConfig `json:"injector,omitempty"`
}

func (c *ResourceConfig) Validate() error {
	if c.GroupVersionKind.Empty() {
		return errors.New("resource is empty")
	}

	for i, dep := range c.Dependents {
		if dep.Empty() {
			return fmt.Errorf("dependents[%d] is empty", i)
		}
	}

	for i, ref := range c.References {
		err := ref.Validate()
		if err != nil {
			return fmt.Errorf("references[%d]: %v", i, err)
		}
	}

	if c.Reconciler != nil {
		err := c.Reconciler.Validate()
		if err != nil {
			return fmt.Errorf("reconciler: %v", err)
		}
	}

	if c.Finalizer != nil {
		err := c.Finalizer.Validate()
		if err != nil {
			return fmt.Errorf("finalizer: %v", err)
		}
	}

	if c.ResyncPeriod != "" {
		_, err := time.ParseDuration(c.ResyncPeriod)
		if err != nil {
			return fmt.Errorf("invalid resync period: %v", err)
		}
	}

	if c.Validator != nil {
		err := c.Validator.Validate()
		if err != nil {
			return fmt.Errorf("validator: %v", err)
		}
	}

	if c.Mutator != nil {
		err := c.Mutator.Validate()
		if err != nil {
			return fmt.Errorf("mutator: %v", err)
		}
	}

	if c.Injector != nil {
		err := c.Injector.Validate()
		if err != nil {
			return fmt.Errorf("injector: %v", err)
		}
	}

	return nil
}

type DependentConfig struct {
	schema.GroupVersionKind
	Orphan bool `json:"orphan"`
}

func (c *DependentConfig) Validate() error {
	if c.GroupVersionKind.Empty() {
		return errors.New("resource is empty")
	}

	return nil
}

type ReferenceConfig struct {
	schema.GroupVersionKind
	NameFieldPath string `json:"nameFieldPath"`
}

func (c *ReferenceConfig) Validate() error {
	if c.GroupVersionKind.Empty() {
		return errors.New("resource is empty")
	}

	if c.NameFieldPath == "" {
		return errors.New("nameFieldPath must be specified")
	}

	return nil
}

type ReconcilerConfig struct {
	HandlerConfig
	RequeueAfter string `json:"requeueAfter"`
	Observe      bool   `json:"observe"`
}

func (c *ReconcilerConfig) Validate() error {
	if c.RequeueAfter != "" {
		_, err := time.ParseDuration(c.RequeueAfter)
		if err != nil {
			return fmt.Errorf("invalid requeueAfter: %v", err)
		}
	}

	return c.HandlerConfig.Validate()
}

type InjectorConfig struct {
	HandlerConfig
	VerifyKeyFile string `json:"verifyKeyFile"`
}

func (c *InjectorConfig) Validate() error {
	if c.VerifyKeyFile == "" {
		return errors.New("verification key file must be specified")
	}

	return c.HandlerConfig.Validate()
}

type HandlerConfig struct {
	Exec *ExecHandlerConfig `json:"exec"`
	HTTP *HTTPHandlerConfig `json:"http"`

	StateHandler            handler.StateHandler            `json:"-"`
	AdmissionRequestHandler handler.AdmissionRequestHandler `json:"-"`
	InjectionRequestHandler handler.InjectionRequestHandler `json:"-"`
}

func (c *HandlerConfig) Validate() error {
	specified := 0

	if c.Exec != nil {
		specified++
	}
	if c.HTTP != nil {
		specified++
	}
	if c.StateHandler != nil || c.AdmissionRequestHandler != nil || c.InjectionRequestHandler != nil {
		specified++
	}

	if specified == 0 {
		return errors.New("handler must be specified")
	}
	if specified > 1 {
		return errors.New("exactly one handler must be specified")
	}

	if c.Exec != nil {
		err := c.Exec.Validate()
		if err != nil {
			return err
		}
	}

	if c.HTTP != nil {
		err := c.HTTP.Validate()
		if err != nil {
			return err
		}
	}

	return nil
}

type ExecHandlerConfig struct {
	Command    string            `json:"command"`
	Args       []string          `json:"args"`
	WorkingDir string            `json:"workingDir"`
	Env        map[string]string `json:"env"`
	Timeout    string            `json:"timeout"`
	Debug      bool              `json:"debug"`
}

func (c ExecHandlerConfig) Validate() error {
	if c.Command == "" {
		return errors.New("command must be specified")
	}

	if c.Timeout != "" {
		_, err := time.ParseDuration(c.Timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout: %v", err)
		}
	}

	return nil
}

type HTTPHandlerConfig struct {
	URL     string     `json:"url"`
	TLS     *TLSConfig `json:"tls,omitempty"`
	Timeout string     `json:"timeout"`
	Debug   bool       `json:"debug"`
}

func (c HTTPHandlerConfig) Validate() error {
	if c.URL == "" {
		return errors.New("url must be specified")
	}

	if c.Timeout != "" {
		_, err := time.ParseDuration(c.Timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout: %v", err)
		}
	}

	return nil
}

type FuncHandlerConfig struct {
	Handler handler.Handler `json:"-"`
}

func (c *FuncHandlerConfig) Validate() error {
	if c.Handler == nil {
		return errors.New("handler must be specified")
	}

	return nil
}

type ServerConfig struct {
	Host string     `json:"host"`
	Port int        `json:"port"`
	TLS  *TLSConfig `json:"tls"`
}

func (c *ServerConfig) Validate() error {
	if c.Port == 0 {
		return errors.New("port must be specified")
	}

	if c.TLS != nil {
		err := c.TLS.Validate()
		if err != nil {
			return fmt.Errorf("tls: %v", err)
		}
	}

	return nil
}

type TLSConfig struct {
	CertFile   string `json:"certFile"`
	KeyFile    string `json:"keyFile"`
	CACertFile string `json:"caCertFile"`
}

func (c *TLSConfig) Validate() error {
	if c.CertFile == "" && c.KeyFile != "" {
		return errors.New("certificate file must be specified")
	}

	if c.CertFile != "" && c.KeyFile == "" {
		return errors.New("certificate key file must be specified")
	}

	return nil
}
