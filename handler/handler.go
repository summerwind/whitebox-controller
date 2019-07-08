package handler

import (
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/summerwind/whitebox-controller/reconciler/state"
	"github.com/summerwind/whitebox-controller/webhook/injection"
)

type Handler interface {
	Run(buf []byte) ([]byte, error)
}

type StateHandler interface {
	HandleState(*state.State) error
}

type AdmissionRequestHandler interface {
	HandleAdmissionRequest(admission.Request) (admission.Response, error)
}

type InjectionRequestHandler interface {
	HandleInjectionRequest(injection.Request) (injection.Response, error)
}
