package handler

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	ActionReconcile = "reconcile"
)

type State struct {
	Action   string                     `json:"action"`
	Resource *unstructured.Unstructured `json:"resource"`
}

func NewState(action string, resource *unstructured.Unstructured) *State {
	return &State{
		Action:   action,
		Resource: resource,
	}
}

type Handler interface {
	Run(req *State) (*State, error)
}
