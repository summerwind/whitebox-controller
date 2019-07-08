package state

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// State is passed to and received from handler.
type State struct {
	Object       *unstructured.Unstructured              `json:"object"`
	Dependents   map[string][]*unstructured.Unstructured `json:"dependents,omitempty"`
	References   map[string][]*unstructured.Unstructured `json:"references,omitempty"`
	Events       []Event                                 `json:"events,omitempty"`
	Requeue      bool                                    `json:"requeue,omitempty"`
	RequeueAfter int                                     `json:"requeueAfter,omitempty"`
}

// NewState returns a new state with specified object.
func New(object *unstructured.Unstructured, deps, refs map[string][]*unstructured.Unstructured) *State {
	return &State{
		Object:     object,
		Dependents: deps,
		References: refs,
		Events:     []Event{},
	}
}

// Copy returns a copy of current state.
func (s *State) Copy() *State {
	ns := &State{
		Object:     s.Object.DeepCopy(),
		Dependents: map[string][]*unstructured.Unstructured{},
		References: map[string][]*unstructured.Unstructured{},
	}

	if len(s.Dependents) > 0 {
		for key, deps := range s.Dependents {
			ns.Dependents[key] = make([]*unstructured.Unstructured, len(s.Dependents[key]))
			for i := range deps {
				ns.Dependents[key][i] = s.Dependents[key][i].DeepCopy()
			}
		}
	}

	if len(s.References) > 0 {
		for key, refs := range s.References {
			ns.References[key] = make([]*unstructured.Unstructured, len(s.References[key]))
			for i := range refs {
				ns.References[key][i] = s.References[key][i].DeepCopy()
			}
		}
	}

	if len(s.Events) > 0 {
		ns.Events = make([]Event, len(s.Events))
		for i := range s.Events {
			ns.Events[i] = s.Events[i]
		}
	}

	return ns
}

// Diff compares two states and returns lists of modified objects.
func (s *State) Diff(ns *State) ([]*unstructured.Unstructured, []*unstructured.Unstructured, []*unstructured.Unstructured) {
	created := []*unstructured.Unstructured{}
	updated := []*unstructured.Unstructured{}
	deleted := []*unstructured.Unstructured{}

	if ns.Object == nil {
		deleted = append(deleted, s.Object)
	} else {
		update := true
		if s.Object.GetNamespace() != ns.Object.GetNamespace() {
			update = false
		}
		if s.Object.GetName() != ns.Object.GetName() {
			update = false
		}
		if reflect.DeepEqual(s.Object, ns.Object) {
			update = false
		}

		if update {
			updated = append(updated, ns.Object)
		}
	}

	checked := map[string]struct{}{}

	// Search for updated or deleted dependent resources.
	for key := range s.Dependents {
		_, ok := ns.Dependents[key]
		if !ok {
			deleted = append(deleted, s.Dependents[key]...)
		}

		for i := range s.Dependents[key] {
			found := false
			dep := s.Dependents[key][i]

			for j := range ns.Dependents[key] {
				newDep := ns.Dependents[key][j]
				if dep.GetNamespace() != newDep.GetNamespace() {
					continue
				}
				if dep.GetName() != newDep.GetName() {
					continue
				}
				if s.Object.GetNamespace() != newDep.GetNamespace() {
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

			ck := fmt.Sprintf("%s/%s/%s", key, dep.GetNamespace(), dep.GetName())
			checked[ck] = struct{}{}
		}
	}

	// Search for new dependent resources.
	for key := range ns.Dependents {
		_, ok := s.Dependents[key]
		if !ok {
			created = append(created, ns.Dependents[key]...)
		}

		for i := range ns.Dependents[key] {
			newDep := ns.Dependents[key][i]

			ck := fmt.Sprintf("%s/%s/%s", key, newDep.GetNamespace(), newDep.GetName())
			_, ok := checked[ck]
			if ok {
				continue
			}

			if s.Object.GetNamespace() != newDep.GetNamespace() {
				continue
			}
			if key != ResourceKey(newDep.GroupVersionKind()) {
				continue
			}

			created = append(created, newDep)
		}
	}

	return created, updated, deleted
}

// Pack parses v and stores the state the value to the state.
func Pack(v interface{}, state *State) error {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to pack: %v", err)
	}

	err = json.Unmarshal(b, state)
	if err != nil {
		return fmt.Errorf("failed to pack: %v", err)
	}

	return nil
}

// Unpack parses the state and stores the value pointed to by v.
func Unpack(state *State, v interface{}) error {
	b, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to unpack: %v", err)
	}

	err = json.Unmarshal(b, v)
	if err != nil {
		return fmt.Errorf("failed to unpack: %v", err)
	}

	return nil
}

func ResourceKey(gvk schema.GroupVersionKind) string {
	if gvk.Group == "" {
		return strings.ToLower(fmt.Sprintf("%s.%s", gvk.Kind, gvk.Version))
	}

	return strings.ToLower(fmt.Sprintf("%s.%s.%s", gvk.Kind, gvk.Version, gvk.Group))
}
