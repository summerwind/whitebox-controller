package reconciler

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Config struct {
	Resource           schema.GroupVersionKind   `json:"resource"`
	DependentResources []schema.GroupVersionKind `json:"dependentResources"`
	Handlers           map[string]Handler        `json:"handlers"`
}

type Handler struct {
	Container corev1.Container `json:"container"`
}
