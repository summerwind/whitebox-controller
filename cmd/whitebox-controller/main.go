package main

import (
	"flag"
	"fmt"
	"os"

	kconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	"github.com/summerwind/whitebox-controller/config"
	"github.com/summerwind/whitebox-controller/controller"
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

	opts := manager.Options{}
	if c.Metrics != nil {
		opts.MetricsBindAddress = fmt.Sprintf("%s:%d", c.Metrics.Host, c.Metrics.Port)
	}

	mgr, err := manager.New(kc, opts)
	if err != nil {
		log.Error(err, "could not create manager")
		os.Exit(1)
	}

	for i, _ := range c.Controllers {
		cc := c.Controllers[i]
		_, err := controller.New(cc, mgr)
		if err != nil {
			log.Error(err, "could not create controller")
			os.Exit(1)
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
