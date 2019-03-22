package handler

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

type Request struct {
	Resource       *unstructured.Unstructured   `json:"resource"`
	OwnedResources []*unstructured.Unstructured `json:"ownedResources"`
}

type Response struct {
	Resource       *unstructured.Unstructured   `json:"resource"`
	OwnedResources []*unstructured.Unstructured `json:"ownedResources"`
}

type ResponseStatus string

const (
	ResponseStatusSuccess ResponseStatus = "success"
	ResponseStatusFailure ResponseStatus = "failure"
	ResponseStatusError   ResponseStatus = "error"
)
