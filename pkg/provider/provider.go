// Package provider is the implementation of custom metric and external metric apis
// see https://github.com/kubernetes/community/blob/master/contributors/design-proposals/instrumentation/custom-metrics-api.md#api-paths
package provider

import (
	"github.com/bbaobelief/k8s-falcon-metrics-adapter/pkg/falcon/custommetrics"
	"github.com/bbaobelief/k8s-falcon-metrics-adapter/pkg/metriccache"
	"github.com/kubernetes-sigs/custom-metrics-apiserver/pkg/provider"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"sync"
)

type FalconProvider struct {
	falconClient custommetrics.FalconClient
	kubeClient   dynamic.Interface
	clientSet    clientset.Interface
	kubeConfig   *rest.Config
	mapper       apimeta.RESTMapper
	valuesLock   sync.RWMutex
	metricCache  *metriccache.MetricCache
}

func NewFalconProvider(mapper apimeta.RESTMapper, kubeClient dynamic.Interface, clientSet clientset.Interface, kubeConfig *rest.Config, falconClient custommetrics.FalconClient, metricCache *metriccache.MetricCache) provider.CustomMetricsProvider {
	return &FalconProvider{
		mapper:       mapper,
		kubeClient:   kubeClient,
		clientSet:    clientSet,
		kubeConfig:   kubeConfig,
		falconClient: falconClient,
		metricCache:  metricCache,
	}
}
