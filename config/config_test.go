package config

import (
	"testing"

	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestConfigValidate(t *testing.T) {
	var (
		err error
		c   *Config
	)

	RegisterTestingT(t)

	// Valid
	c = newTestConfig()
	err = c.Validate()
	Expect(err).NotTo(HaveOccurred())

	// No resources
	c = newTestConfig()
	c.Resources = []*ResourceConfig{}
	err = c.Validate()
	Expect(err).To(HaveOccurred())

	// Invalid resource
	c = newTestConfig()
	c.Resources[0].GroupVersionKind = schema.GroupVersionKind{}
	err = c.Validate()
	Expect(err).To(HaveOccurred())

	// Invalid webhook
	c = newTestConfig()
	c.Webhook.Port = 0
	err = c.Validate()
	Expect(err).To(HaveOccurred())

	// Invalid metrics
	c = newTestConfig()
	c.Metrics.Port = 0
	err = c.Validate()
	Expect(err).To(HaveOccurred())
}

func TestResourceConfigValidate(t *testing.T) {
	var (
		err error
		c   *ResourceConfig
	)

	RegisterTestingT(t)

	// Valid
	c = newTestConfig().Resources[0]
	err = c.Validate()
	Expect(err).NotTo(HaveOccurred())

	// Invalid GVK
	c = newTestConfig().Resources[0]
	c.GroupVersionKind = schema.GroupVersionKind{}
	err = c.Validate()
	Expect(err).To(HaveOccurred())

	// Invalid dependents
	c = newTestConfig().Resources[0]
	c.Dependents[0].GroupVersionKind = schema.GroupVersionKind{}
	err = c.Validate()
	Expect(err).To(HaveOccurred())

	// Invalid references
	c = newTestConfig().Resources[0]
	c.References[0].GroupVersionKind = schema.GroupVersionKind{}
	err = c.Validate()
	Expect(err).To(HaveOccurred())

	// Invalid reconciler
	c = newTestConfig().Resources[0]
	c.Reconciler.HandlerConfig.Exec = nil
	err = c.Validate()
	Expect(err).To(HaveOccurred())

	// Invalid finalizer
	c = newTestConfig().Resources[0]
	c.Finalizer.Exec = nil
	err = c.Validate()
	Expect(err).To(HaveOccurred())

	// Invalid resync period
	c = newTestConfig().Resources[0]
	c.ResyncPeriod = "invalid"
	err = c.Validate()
	Expect(err).To(HaveOccurred())

	// Invalid validator
	c = newTestConfig().Resources[0]
	c.Validator.Exec = nil
	err = c.Validate()
	Expect(err).To(HaveOccurred())

	// Invalid mutator
	c = newTestConfig().Resources[0]
	c.Mutator.Exec = nil
	err = c.Validate()
	Expect(err).To(HaveOccurred())

	// Invalid injector
	c = newTestConfig().Resources[0]
	c.Injector.Exec = nil
	err = c.Validate()
	Expect(err).To(HaveOccurred())
}

func TestDependentConfigValidate(t *testing.T) {
	var (
		err error
		c   DependentConfig
	)

	RegisterTestingT(t)

	// Valid
	c = newTestConfig().Resources[0].Dependents[0]
	err = c.Validate()
	Expect(err).NotTo(HaveOccurred())

	// Empty GVK
	c = newTestConfig().Resources[0].Dependents[0]
	c.GroupVersionKind = schema.GroupVersionKind{}
	err = c.Validate()
	Expect(err).To(HaveOccurred())
}

func TestReferenceConfigValidate(t *testing.T) {
	var (
		err error
		c   ReferenceConfig
	)

	RegisterTestingT(t)

	// Valid
	c = newTestConfig().Resources[0].References[0]
	err = c.Validate()
	Expect(err).NotTo(HaveOccurred())

	// Empty GVK
	c = newTestConfig().Resources[0].References[0]
	c.GroupVersionKind = schema.GroupVersionKind{}
	err = c.Validate()
	Expect(err).To(HaveOccurred())

	// Empty name field path
	c = newTestConfig().Resources[0].References[0]
	c.NameFieldPath = ""
	err = c.Validate()
	Expect(err).To(HaveOccurred())
}

func TestReconcilerConfigValidate(t *testing.T) {
	var (
		err error
		c   *ReconcilerConfig
	)

	RegisterTestingT(t)

	// Valid
	c = newTestConfig().Resources[0].Reconciler
	err = c.Validate()
	Expect(err).NotTo(HaveOccurred())

	// Invalid requeue after
	c = newTestConfig().Resources[0].Reconciler
	c.RequeueAfter = "invalid"
	err = c.Validate()
	Expect(err).To(HaveOccurred())

	// Invalid handler
	c = newTestConfig().Resources[0].Reconciler
	c.HandlerConfig.Exec = nil
	err = c.Validate()
	Expect(err).To(HaveOccurred())
}

func TestInjectorConfigValidate(t *testing.T) {
	var (
		err error
		c   *InjectorConfig
	)

	RegisterTestingT(t)

	// Valid
	c = newTestConfig().Resources[0].Injector
	err = c.Validate()
	Expect(err).NotTo(HaveOccurred())

	// Invalid verification key file
	c = newTestConfig().Resources[0].Injector
	c.VerifyKeyFile = ""
	err = c.Validate()
	Expect(err).To(HaveOccurred())

	// Invalid handler
	c = newTestConfig().Resources[0].Injector
	c.HandlerConfig.Exec = nil
	err = c.Validate()
	Expect(err).To(HaveOccurred())
}

func TestHandlerConfig(t *testing.T) {
	var (
		err error
		c   *HandlerConfig
	)

	RegisterTestingT(t)

	// Valid
	c = &HandlerConfig{
		Exec: &ExecHandlerConfig{
			Command:    "/bin/controller",
			Args:       []string{"command"},
			WorkingDir: "",
			Env:        map[string]string{},
			Timeout:    "30s",
			Debug:      true,
		},
	}
	err = c.Validate()
	Expect(err).NotTo(HaveOccurred())

	// No handlers
	c = &HandlerConfig{}
	err = c.Validate()
	Expect(err).To(HaveOccurred())

	// Multiple handlers
	c = &HandlerConfig{
		Exec: &ExecHandlerConfig{},
		HTTP: &HTTPHandlerConfig{},
	}
	err = c.Validate()
	Expect(err).To(HaveOccurred())

	// Invalid exec handler
	c = &HandlerConfig{
		Exec: &ExecHandlerConfig{},
	}
	err = c.Validate()
	Expect(err).To(HaveOccurred())

	// Invalid HTTP handler
	c = &HandlerConfig{
		HTTP: &HTTPHandlerConfig{},
	}
	err = c.Validate()
	Expect(err).To(HaveOccurred())
}

func TestExecHandlerConfig(t *testing.T) {
	var (
		err error
		c   *ExecHandlerConfig
	)

	// Valid
	c = &ExecHandlerConfig{
		Command:    "/bin/controller",
		Args:       []string{"command"},
		WorkingDir: "",
		Env:        map[string]string{},
		Timeout:    "30s",
		Debug:      true,
	}
	err = c.Validate()
	Expect(err).NotTo(HaveOccurred())

	// Invalid command
	c = &ExecHandlerConfig{
		Command: "",
		Timeout: "30s",
	}
	err = c.Validate()
	Expect(err).To(HaveOccurred())

	// Invalid timeout
	c = &ExecHandlerConfig{
		Command: "/bin/controller",
		Timeout: "invalid",
	}
	err = c.Validate()
	Expect(err).To(HaveOccurred())
}

func TestHTTPHandlerConfig(t *testing.T) {
	var (
		err error
		c   *HTTPHandlerConfig
	)

	// Valid
	c = &HTTPHandlerConfig{
		URL: "http://127.0.0.1:8080",
		TLS: &TLSConfig{
			CACertFile: "ca.pem",
		},
		Timeout: "30s",
		Debug:   true,
	}
	err = c.Validate()
	Expect(err).NotTo(HaveOccurred())

	// Invalid URL
	c = &HTTPHandlerConfig{
		URL:     "",
		Timeout: "30s",
		Debug:   true,
	}
	err = c.Validate()
	Expect(err).To(HaveOccurred())

	// Invalid timeout
	c = &HTTPHandlerConfig{
		URL:     "http://127.0.0.1:8080",
		Timeout: "invalid",
		Debug:   true,
	}
	err = c.Validate()
	Expect(err).To(HaveOccurred())
}

func TestServerConfig(t *testing.T) {
	var (
		err error
		c   *ServerConfig
	)

	// Valid
	c = &ServerConfig{
		Host: "127.0.0.1",
		Port: 443,
		TLS: &TLSConfig{
			CertFile: "server.pem",
			KeyFile:  "server-key.pem",
		},
	}
	err = c.Validate()
	Expect(err).NotTo(HaveOccurred())

	// Invalid port
	c = &ServerConfig{
		Host: "127.0.0.1",
		TLS: &TLSConfig{
			CertFile: "server.pem",
			KeyFile:  "server-key.pem",
		},
	}
	err = c.Validate()
	Expect(err).To(HaveOccurred())

	// Invalid TLS config
	c = &ServerConfig{
		Host: "127.0.0.1",
		Port: 443,
		TLS: &TLSConfig{
			CertFile: "server.pem",
		},
	}
	err = c.Validate()
	Expect(err).To(HaveOccurred())
}

func TestTLSConfig(t *testing.T) {
	var (
		err error
		c   *TLSConfig
	)

	// Valid
	c = &TLSConfig{
		CertFile: "server.pem",
		KeyFile:  "server-key.pem",
	}
	err = c.Validate()
	Expect(err).NotTo(HaveOccurred())

	// Invalid certificate file
	c = &TLSConfig{
		KeyFile: "server-key.pem",
	}
	err = c.Validate()
	Expect(err).To(HaveOccurred())

	// Invalid certificate key file
	c = &TLSConfig{
		CertFile: "server.pem",
	}
	err = c.Validate()
	Expect(err).To(HaveOccurred())
}

func newTestConfig() *Config {
	return &Config{
		Resources: []*ResourceConfig{
			{
				GroupVersionKind: schema.GroupVersionKind{
					Group:   "example.com",
					Version: "v1alpha1",
					Kind:    "Test",
				},
				Dependents: []DependentConfig{
					DependentConfig{
						GroupVersionKind: schema.GroupVersionKind{
							Group:   "example.org",
							Version: "v1alpha1",
							Kind:    "ResourceA",
						},
						Orphan: false,
					},
					DependentConfig{
						GroupVersionKind: schema.GroupVersionKind{
							Group:   "example.org",
							Version: "v1alpha1",
							Kind:    "ResourceB",
						},
						Orphan: false,
					},
				},
				References: []ReferenceConfig{
					ReferenceConfig{
						GroupVersionKind: schema.GroupVersionKind{
							Group:   "example.org",
							Version: "v1alpha1",
							Kind:    "ResourceX",
						},
						NameFieldPath: ".spec.x",
					},
					ReferenceConfig{
						GroupVersionKind: schema.GroupVersionKind{
							Group:   "example.org",
							Version: "v1alpha1",
							Kind:    "ResourceY",
						},
						NameFieldPath: ".spec.y",
					},
				},
				Reconciler: &ReconcilerConfig{
					HandlerConfig: HandlerConfig{
						Exec: &ExecHandlerConfig{
							Command:    "/bin/controller",
							Args:       []string{"reconcile"},
							WorkingDir: "",
							Env:        map[string]string{},
							Timeout:    "30s",
							Debug:      true,
						},
					},
					RequeueAfter: "30s",
					Observe:      false,
				},
				Finalizer: &HandlerConfig{
					Exec: &ExecHandlerConfig{
						Command:    "/bin/controller",
						Args:       []string{"finalize"},
						WorkingDir: "",
						Env:        map[string]string{},
						Timeout:    "30s",
						Debug:      true,
					},
				},
				ResyncPeriod: "60m",
				Validator: &HandlerConfig{
					Exec: &ExecHandlerConfig{
						Command:    "/bin/controller",
						Args:       []string{"validate"},
						WorkingDir: "",
						Env:        map[string]string{},
						Timeout:    "30s",
						Debug:      true,
					},
				},
				Mutator: &HandlerConfig{
					Exec: &ExecHandlerConfig{
						Command:    "/bin/controller",
						Args:       []string{"mutate"},
						WorkingDir: "",
						Env:        map[string]string{},
						Timeout:    "30s",
						Debug:      true,
					},
				},
				Injector: &InjectorConfig{
					HandlerConfig: HandlerConfig{
						Exec: &ExecHandlerConfig{
							Command:    "/bin/controller",
							Args:       []string{"inject"},
							WorkingDir: "",
							Env:        map[string]string{},
							Timeout:    "30s",
							Debug:      true,
						},
					},
					VerifyKeyFile: "verify-key.pem",
				},
			},
		},
		Webhook: &ServerConfig{
			Host: "127.0.0.1",
			Port: 443,
			TLS: &TLSConfig{
				CertFile: "server.pem",
				KeyFile:  "server-key.pem",
			},
		},
		Metrics: &ServerConfig{
			Host: "127.0.0.1",
			Port: 91438,
		},
	}
}
