package reconciler

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/summerwind/whitebox-controller/config"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type State struct {
	Object     *unstructured.Unstructured              `json:"object"`
	Dependents map[string][]*unstructured.Unstructured `json:"dependents"`
	References map[string][]*unstructured.Unstructured `json:"references"`
	Events     []StateEvent                            `json:"events"`
}

func NewState(object *unstructured.Unstructured, deps, refs map[string][]*unstructured.Unstructured) *State {
	return &State{
		Object:     object,
		Dependents: deps,
		References: refs,
		Events:     []StateEvent{},
	}
}

func (s *State) Validate(new *State, c *config.ControllerConfig) error {
	if new.Object != nil {
		namespace := new.Object.GetNamespace()

		if !reflect.DeepEqual(new.Object.GroupVersionKind(), s.Object.GroupVersionKind()) {
			return errors.New("resource: group/version/kind does not match")
		}
		if namespace != s.Object.GetNamespace() {
			return errors.New("resource: namespace does not match")
		}
		if new.Object.GetName() != s.Object.GetName() {
			return errors.New("resource: name does not match")
		}

		for key := range new.Dependents {
			for i, dep := range new.Dependents[key] {
				if dep.GetNamespace() != namespace {
					return fmt.Errorf("dependents[%s][%d]: namespace does not match", key, i)
				}
			}
		}
	}

	for key := range new.Dependents {
		if len(c.Dependents) == 0 {
			return errors.New("no dependents specified in the configuration")
		}

		matched := false
		for _, gvk := range c.Dependents {
			if key == getKindArg(gvk) {
				matched = true
				break
			}
		}

		if !matched {
			return fmt.Errorf("dependents[%s]: unexpected group/version/kind", key)
		}
	}

	return nil
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
					updated = append(updated, dep)
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

func getKindArg(gvk schema.GroupVersionKind) string {
	if gvk.Group == "" {
		return strings.ToLower(fmt.Sprintf("%s.%s", gvk.Kind, gvk.Version))
	}

	return strings.ToLower(fmt.Sprintf("%s.%s.%s", gvk.Kind, gvk.Version, gvk.Group))
}
