package main

import (
	"flag"
	"os"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/component-base/logs"

	clientset "github.com/bbaobelief/k8s-falcon-metrics-adapter/pkg/client/clientset/versioned"
	informers "github.com/bbaobelief/k8s-falcon-metrics-adapter/pkg/client/informers/externalversions"
	"github.com/bbaobelief/k8s-falcon-metrics-adapter/pkg/controller"
	"github.com/bbaobelief/k8s-falcon-metrics-adapter/pkg/falcon/custommetrics"
	"github.com/bbaobelief/k8s-falcon-metrics-adapter/pkg/metriccache"
	falconrovider "github.com/bbaobelief/k8s-falcon-metrics-adapter/pkg/provider"
	basecmd "github.com/kubernetes-sigs/custom-metrics-apiserver/pkg/cmd"
	"k8s.io/klog/v2"
)

func setupFalconProvider(cmd *basecmd.AdapterBase, metricsCache *metriccache.MetricCache) {
	mapper, err := cmd.RESTMapper()
	if err != nil {
		klog.Fatalf("unable to construct discovery REST mapper: %v", err)
	}

	dynamicClient, err := cmd.DynamicClient()
	if err != nil {
		klog.Fatalf("unable to construct dynamic k8s client: %v", err)
	}

	// creates the kubeClientSet
	config, err := cmd.ClientConfig()
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("unable to construct clientSet: %v", err)
	}

	customMetricsClient := custommetrics.NewClient()

	provider := falconrovider.NewFalconProvider(mapper, dynamicClient, clientSet, config, customMetricsClient, metricsCache)
	cmd.WithCustomMetrics(provider)
}

func newController(cmd *basecmd.AdapterBase, metricsCache *metriccache.MetricCache) (*controller.Controller, informers.SharedInformerFactory) {
	clientConfig, err := cmd.ClientConfig()
	if err != nil {
		klog.Fatalf("unable to construct client config: %s", err)
	}
	adapterClientSet, err := clientset.NewForConfig(clientConfig)
	if err != nil {
		klog.Fatalf("unable to construct lister client to initialize provider: %v", err)
	}

	adapterInformerFactory := informers.NewSharedInformerFactory(adapterClientSet, time.Second*30)
	handler := controller.NewHandler(adapterInformerFactory.Metrics().V1alpha1().CustomMetrics().Lister(), metricsCache)

	ctl := controller.NewController(adapterInformerFactory.Metrics().V1alpha1().CustomMetrics(), &handler)

	return ctl, adapterInformerFactory
}

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	cmd := &basecmd.AdapterBase{}
	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	cmd.Flags().Parse(os.Args)

	stopCh := make(chan struct{})
	defer close(stopCh)

	cache := metriccache.NewMetricCache()

	// start and run contoller components
	ctl, adapterInformerFactory := newController(cmd, cache)
	go adapterInformerFactory.Start(stopCh)
	go ctl.Run(2, time.Second, stopCh)

	//setup and run metric server
	setupFalconProvider(cmd, cache)

	klog.Info("falcon metrics adapter started.")

	if err := cmd.Run(stopCh); err != nil {
		klog.Fatalf("Unable to run Falcon metrics adapter: %v", err)
	}
}
