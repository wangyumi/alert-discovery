package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/pflag"

	k8s_client "k8s.io/kubernetes/pkg/client/unversioned"
	kubectl_util "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

const (
	healthPort = 23333
)

var (
	flags     = pflag.NewFlagSet("", pflag.ExitOnError)
	inCluster = flags.Bool("running-in-cluster", false,
		`Optional, if this controller is running in a kubernetes cluster, use the
		pod secrets for creating a Kubernetes client.`)
	healthzPort   = flags.Int("healthz-port", healthPort, "port for healthz endpoint.")
	configmapNs   = flags.String("configmap-ns", "", "namespace of configmap")
	configmapName = flags.String("configmap-name", "", "configmap to put generated rule in")
)

func main() {
	var client *k8s_client.Client
	flags.AddGoFlagSet(flag.CommandLine)
	flags.Parse(os.Args)
	clientConfig := kubectl_util.DefaultClientConfig(flags)

	// Workaround of noisy log, see https://github.com/kubernetes/kubernetes/issues/17162
	flag.CommandLine.Parse([]string{})

	var err error
	if *inCluster {
		client, err = k8s_client.NewInCluster()
	} else {
		config, connErr := clientConfig.ClientConfig()
		if connErr != nil {
			glog.Fatalf("error connecting to the client: %v", err)
		}
		client, err = k8s_client.New(config)
	}
	if err != nil {
		glog.Fatalf("failed to create client: %v", err)
	}

	if *configmapNs == "" || *configmapName == "" {
		glog.Fatalf("not enough config: configmap-ns: %s, configmap-name: %s", *configmapNs, *configmapName)
	}

	controller, err := newController(client, *configmapNs, *configmapName)
	if err != nil {
		glog.Fatalf("error create controller: %v", err)
	}

	go registerHandlers()
	go handleSigterm(controller)

	controller.Run()

	for {
		glog.Infof("Handle quit, awaiting...")
		time.Sleep(30 * time.Second)
	}
}

func handleSigterm(cc *controller) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM)
	<-signalChan
	glog.Infof("Received SIGTERM, shutting down")

	exitCode := 0
	if err := cc.Stop(); err != nil {
		glog.Infof("Error during shutdown %v", err)
		exitCode = 1
	}

	glog.Infof("Exiting with %v", exitCode)
	os.Exit(exitCode)
}

func registerHandlers() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%v", *healthzPort),
		Handler: mux,
	}
	glog.Fatal(server.ListenAndServe())
}
