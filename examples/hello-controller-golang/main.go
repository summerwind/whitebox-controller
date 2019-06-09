package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	"github.com/summerwind/whitebox-controller/config"
	"github.com/summerwind/whitebox-controller/manager"
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

type HelloHandler struct{}

func (h HelloHandler) Run(buf []byte) ([]byte, error) {
	state := State{}

	err := json.Unmarshal(buf, &state)
	if err != nil {
		return nil, err
	}

	if state.Object.Status.Phase != "completed" {
		log.Printf("message: %s", state.Object.Spec.Message)
		state.Object.Status.Phase = "completed"
	}

	out, err := json.Marshal(&state)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func main() {
	handler := HelloHandler{}

	c := &config.Config{
		Controllers: []*config.ControllerConfig{
			&config.ControllerConfig{
				Name: "hello-controller",
				Resource: schema.GroupVersionKind{
					Group:   "whitebox.summerwind.github.io",
					Version: "v1alpha1",
					Kind:    "Hello",
				},
				Reconciler: &config.ReconcilerConfig{
					HandlerConfig: config.HandlerConfig{
						Func: &config.FuncHandlerConfig{
							Handler: handler,
						},
					},
				},
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
