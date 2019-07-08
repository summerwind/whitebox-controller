package state

import "errors"

// Event represents an event of Kubernetes.
type Event struct {
	Type    string `json:"type"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// Validate validates the content of event.
func (e *Event) Validate() error {
	if e.Type == "" {
		return errors.New("type must be specified")
	}

	return nil
}
