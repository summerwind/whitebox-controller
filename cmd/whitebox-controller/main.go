package main

import (
	"flag"
	"fmt"
	"os"

	kconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	"github.com/summerwind/whitebox-controller/config"
	"github.com/summerwind/whitebox-controller/manager"
)

var (
	VERSION string = "dev"
	COMMIT  string = "HEAD"
)

func main() {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("whitebox-controller")

	var (
		configPath = flag.String("c", "config.yaml", "Path to configuration file")
		version    = flag.Bool("version", false, "Display version information and exit")
	)

	flag.Parse()

	if *version {
		fmt.Printf("%s (%s)\n", VERSION, COMMIT)
		return
	}

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

	mgr, err := manager.New(c, kc)
	if err != nil {
		log.Error(err, "could not create controller manager")
		os.Exit(1)
	}

	err = mgr.Start(signals.SetupSignalHandler())
	if err != nil {
		log.Error(err, "could not start controller manager")
		os.Exit(1)
	}
}
