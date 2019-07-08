package syncer

import (
	"context"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/summerwind/whitebox-controller/config"
)

var log = logf.Log.WithName("syncer")

type Syncer struct {
	client.Client
	C        chan event.GenericEvent
	config   *config.ResourceConfig
	interval time.Duration
}

func New(c *config.ResourceConfig, mgr manager.Manager) (*Syncer, error) {
	interval, err := time.ParseDuration(c.ResyncPeriod)
	if err != nil {
		return nil, fmt.Errorf("invalid resync period: %v", err)
	}

	s := &Syncer{
		Client:   mgr.GetClient(),
		C:        make(chan event.GenericEvent),
		config:   c,
		interval: interval,
	}

	return s, mgr.Add(s)
}

func (s *Syncer) Start(stop <-chan struct{}) error {
	t := time.NewTicker(s.interval)

	name := fmt.Sprintf("%s-controller", strings.ToLower(s.config.Kind))

	for {
		select {
		case <-t.C:
			err := s.Sync()
			if err != nil {
				log.Error(err, "Sync error", "syncer", name)
			}
			log.Info("Synced", "syncer", name)
		case <-stop:
			log.Info("Stopping syncer", "syncer", name)
			return nil
		}
	}
}

func (s *Syncer) Sync() error {
	instanceList := &unstructured.UnstructuredList{}
	instanceList.SetGroupVersionKind(s.config.GroupVersionKind)

	err := s.List(context.TODO(), instanceList)
	if err != nil {
		return err
	}

	for _, instance := range instanceList.Items {
		s.C <- event.GenericEvent{
			Meta: &metav1.ObjectMeta{
				Name:      instance.GetName(),
				Namespace: instance.GetNamespace(),
			},
		}
	}

	return nil
}
