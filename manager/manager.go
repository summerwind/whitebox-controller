package manager

import (
	"fmt"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/summerwind/whitebox-controller/config"
	"github.com/summerwind/whitebox-controller/controller"
	"github.com/summerwind/whitebox-controller/webhook"
)

func New(c *config.Config, kc *rest.Config) (manager.Manager, error) {
	err := c.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %v", err)
	}

	mgr, err := manager.New(kc, manager.Options{})
	if err != nil {
		return nil, err
	}

	wh := false
	for _, r := range c.Resources {
		if r.Reconciler != nil {
			_, err := controller.New(r, mgr)
			if err != nil {
				return nil, err
			}
		}

		if r.Validator != nil || r.Mutator != nil || r.Injector != nil {
			wh = true
		}
	}

	if wh {
		server, err := webhook.NewServer(c.Webhook, mgr)
		if err != nil {
			return nil, err
		}

		for _, r := range c.Resources {
			if r.Validator != nil {
				server.AddValidator(r)
			}

			if r.Mutator != nil {
				server.AddMutator(r)
			}

			if r.Injector != nil {
				server.AddInjector(r)
			}
		}
	}

	return mgr, nil
}
