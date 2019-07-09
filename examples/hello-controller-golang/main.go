package main

import (
	"fmt"
	"log"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	"github.com/summerwind/whitebox-controller/config"
	"github.com/summerwind/whitebox-controller/manager"
	"github.com/summerwind/whitebox-controller/reconciler/state"
	"github.com/summerwind/whitebox-controller/webhook"
)

type Hello struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HelloSpec   `json:"spec"`
	Status HelloStatus `json:"status"`
}

type HelloSpec struct {
	Message string `json:"message"`
}

type HelloStatus struct {
	Phase string `json:"phase"`
}

type State struct {
	Object Hello `json:"object"`
}

type Handler struct{}

func (h *Handler) HandleState(ss *state.State) error {
	s := State{}

	err := state.Unpack(ss, &s)
	if err != nil {
		return err
	}

	if s.Object.Status.Phase != "completed" {
		log.Printf("message: %s", s.Object.Spec.Message)
		s.Object.Status.Phase = "completed"
	}

	err = state.Pack(&s, ss)
	if err != nil {
		return err
	}

	return nil
}

func (h *Handler) HandleAdmissionRequest(req admission.Request) (admission.Response, error) {
	hello := &Hello{}

	err := webhook.Unpack(req.Object, hello)
	if err != nil {
		return admission.Response{}, err
	}

	if hello.Spec.Message == "" {
		return admission.Denied("message must be specified"), nil
	}

	return admission.Allowed(""), nil
}

func main() {
	logf.SetLogger(logf.ZapLogger(false))

	c := &config.Config{
		Resources: []*config.ResourceConfig{
			{
				GroupVersionKind: schema.GroupVersionKind{
					Group:   "whitebox.summerwind.dev",
					Version: "v1alpha1",
					Kind:    "Hello",
				},
				Reconciler: &config.ReconcilerConfig{
					HandlerConfig: config.HandlerConfig{
						StateHandler: &Handler{},
					},
				},
				Validator: &config.HandlerConfig{
					AdmissionRequestHandler: &Handler{},
				},
			},
		},
		Webhook: &config.ServerConfig{
			Host: "0.0.0.0",
			Port: 443,
			TLS: &config.TLSConfig{
				CertFile: "/etc/tls/tls.crt",
				KeyFile:  "/etc/tls/tls.key",
			},
		},
	}

	kc, err := kconfig.GetConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load kubeconfig: %s\n", err)
		os.Exit(1)
	}

	mgr, err := manager.New(c, kc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create controller manager: %s\n", err)
		os.Exit(1)
	}

	err = mgr.Start(signals.SetupSignalHandler())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start controller manager: %s\n", err)
		os.Exit(1)
	}
}
