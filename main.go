package main

import (
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
	"sigs.k8s.io/controller-runtime/pkg/source"

	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	"github.com/summerwind/whitebox-controller/config"
	"github.com/summerwind/whitebox-controller/observer"
	"github.com/summerwind/whitebox-controller/reconciler"
	"github.com/summerwind/whitebox-controller/syncer"
	"github.com/summerwind/whitebox-controller/webhook"
)

func main() {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("whitebox-controller")

	var configPath = flag.String("c", "config.yaml", "Path to configuration file")

	flag.Parse()

	c, err := config.LoadFile(*configPath)
	if err != nil {
		log.Error(err, "could not load configuration file")
		os.Exit(1)
	}

	kc, err := kconfig.GetConfig()
	if err != nil {
		log.Error(err, "could not load kubernetes configuration")
		os.Exit(1)
	}

	mgr, err := manager.New(kc, manager.Options{})
	if err != nil {
		log.Error(err, "could not create manager")
		os.Exit(1)
	}

	for i, _ := range c.Controllers {
		cc := c.Controllers[i]

		var (
			r       reconcile.Reconciler
			err     error
			observe bool
		)

		if cc.Reconciler != nil {
			r, err = reconciler.New(cc)
			if err != nil {
				log.Error(err, "could not create reconciler")
				os.Exit(1)
			}
		}

		if cc.Observer != nil {
			r, err = observer.New(cc)
			if err != nil {
				log.Error(err, "could not create observer")
				os.Exit(1)
			}
			observe = true
		}

		if r == nil {
			log.Error(err, "either reconciler or observer must be specified")
		}

		ctrl, err := controller.New(cc.Name, mgr, controller.Options{Reconciler: r})
		if err != nil {
			log.Error(err, "could not create controller")
			os.Exit(1)
		}

		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(cc.Resource)

		err = ctrl.Watch(&source.Kind{Type: obj}, &handler.EnqueueRequestForObject{})
		if err != nil {
			log.Error(err, "failed to watch resource")
			os.Exit(1)
		}

		// No need to setup deps and syncer for observer.
		if observe {
			continue
		}

		for _, dep := range cc.Dependents {
			depObj := &unstructured.Unstructured{}
			depObj.SetGroupVersionKind(dep)

			err = ctrl.Watch(&source.Kind{Type: depObj}, &handler.EnqueueRequestForOwner{
				IsController: true,
				OwnerType:    obj,
			})
			if err != nil {
				log.Error(err, "failed to watch dependent resource")
				os.Exit(1)
			}
		}

		if c.Controllers[i].Syncer != nil {
			s, err := syncer.New(cc, mgr)
			if err != nil {
				log.Error(err, "could not create syncer")
				os.Exit(1)
			}

			err = ctrl.Watch(&source.Channel{Source: s.C}, &handler.EnqueueRequestForObject{})
			if err != nil {
				log.Error(err, "failed to watch sync channel")
				os.Exit(1)
			}
		}
	}

	if c.Webhook != nil {
		_, err := webhook.NewServer(c.Webhook, mgr)
		if err != nil {
			log.Error(err, "could not create webhook server")
			os.Exit(1)
		}
	}

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "could not start manager")
		os.Exit(1)
	}
}
