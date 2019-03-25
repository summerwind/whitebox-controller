package reconciler

import (
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type State struct {
	Resource   *unstructured.Unstructured  `json:"resource"`
	Dependents []unstructured.Unstructured `json:"dependents"`
}

func NewState(resource *unstructured.Unstructured, dependents []unstructured.Unstructured) *State {
	return &State{
		Resource:   resource,
		Dependents: dependents,
	}
}

func (s *State) Diff(new *State) ([]unstructured.Unstructured, []unstructured.Unstructured, []unstructured.Unstructured) {
	created := []unstructured.Unstructured{}
	updated := []unstructured.Unstructured{}
	deleted := []unstructured.Unstructured{}

	if new.Resource == nil {
		deleted = append(deleted, *new.Resource)
	} else if !reflect.DeepEqual(s.Resource, new.Resource) {
		updated = append(updated, *new.Resource)
	}

	for _, dep := range s.Dependents {
		found := false

		for _, newDep := range new.Dependents {
			if dep.GetSelfLink() != newDep.GetSelfLink() {
				continue
			}

			found = true
			if !reflect.DeepEqual(dep, newDep) {
				updated = append(updated, dep)
			}
			break
		}

		if !found {
			deleted = append(deleted, dep)
		}
	}

	for _, newDep := range new.Dependents {
		if newDep.GetSelfLink() == "" {
			created = append(created, newDep)
		}
	}

	return created, updated, deleted
}
