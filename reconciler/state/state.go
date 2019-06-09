package state

import (
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type State struct {
	Object     *unstructured.Unstructured              `json:"object"`
	Dependents map[string][]*unstructured.Unstructured `json:"dependents"`
	References map[string][]*unstructured.Unstructured `json:"references"`
	Events     []StateEvent                            `json:"events"`
}

func New(object *unstructured.Unstructured, deps, refs map[string][]*unstructured.Unstructured) *State {
	return &State{
		Object:     object,
		Dependents: deps,
		References: refs,
		Events:     []StateEvent{},
	}
}

func (s *State) Diff(new *State) ([]*unstructured.Unstructured, []*unstructured.Unstructured, []*unstructured.Unstructured) {
	created := []*unstructured.Unstructured{}
	updated := []*unstructured.Unstructured{}
	deleted := []*unstructured.Unstructured{}

	if new.Object == nil {
		deleted = append(deleted, new.Object)
	} else if !reflect.DeepEqual(s.Object, new.Object) {
		updated = append(updated, new.Object)
	}

	for key := range s.Dependents {
		_, ok := new.Dependents[key]
		if !ok {
			deleted = append(deleted, s.Dependents[key]...)
		}

		for i := range s.Dependents[key] {
			found := false
			dep := s.Dependents[key][i]

			for j := range new.Dependents[key] {
				newDep := new.Dependents[key][j]
				if dep.GetSelfLink() != newDep.GetSelfLink() {
					continue
				}

				found = true
				if !reflect.DeepEqual(dep, newDep) {
					updated = append(updated, newDep)
				}
				break
			}

			if !found {
				deleted = append(deleted, dep)
			}
		}
	}

	for key := range new.Dependents {
		_, ok := s.Dependents[key]
		if !ok {
			created = append(created, new.Dependents[key]...)
		}

		for i := range new.Dependents[key] {
			newDep := new.Dependents[key][i]
			if newDep.GetSelfLink() == "" {
				created = append(created, newDep)
			}
		}
	}

	return created, updated, deleted
}

type StateEvent struct {
	Type    string `json:"type"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

func (e *StateEvent) Empty() bool {
	if e.Type == "" || e.Reason == "" || e.Message == "" {
		return true
	}
	return false
}
