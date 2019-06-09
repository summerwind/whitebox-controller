package handler

import (
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/summerwind/whitebox-controller/reconciler/state"
	"github.com/summerwind/whitebox-controller/webhook/injection"
)

type Handler interface {
	Reconcile(*state.State) (*state.State, error)
	Finalize(*state.State) (*state.State, error)

	Validate(*admission.Request) (*admission.Response, error)
	Mutate(*admission.Request) (*admission.Response, error)
	Inject(*injection.Request) (*injection.Response, error)
}
