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
	opts := manager.Options{}
	if c.Metrics != nil {
		opts.MetricsBindAddress = fmt.Sprintf("%s:%d", c.Metrics.Host, c.Metrics.Port)
	}

	mgr, err := manager.New(kc, opts)
	if err != nil {
		return nil, err
	}

	for i, _ := range c.Controllers {
		cc := c.Controllers[i]
		_, err := controller.New(cc, mgr)
		if err != nil {
			return nil, err
		}
	}

	if c.Webhook != nil {
		_, err := webhook.NewServer(c.Webhook, mgr)
		if err != nil {
			return nil, err
		}
	}

	return mgr, nil
}
