package controller

import (
	"fmt"

	listers "github.com/bbaobelief/k8s-falcon-metrics-adapter/pkg/client/listers/metrics/v1alpha1"
	"github.com/bbaobelief/k8s-falcon-metrics-adapter/pkg/falcon/custommetrics"
	"github.com/bbaobelief/k8s-falcon-metrics-adapter/pkg/metriccache"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

// Handler processes the events from the controler for external metrics
type Handler struct {
	metriccache        *metriccache.MetricCache
	customMetricLister listers.CustomMetricLister
}

// NewHandler created a new handler
func NewHandler(customMetricLister listers.CustomMetricLister, metricCache *metriccache.MetricCache) Handler {
	return Handler{
		customMetricLister: customMetricLister,
		metriccache:        metricCache,
	}
}

type ControllerHandler interface {
	Process(queueItem namespacedQueueItem) error
}

// Process validates the item exists then stores updates the metric cached used to make requests to azure
func (h *Handler) Process(queueItem namespacedQueueItem) error {
	ns, name, err := cache.SplitMetaNamespaceKey(queueItem.namespaceKey)
	if err != nil {
		// not a valid key do not put back on queue
		runtime.HandleError(fmt.Errorf("expected namespace/name key in workqueue but got %s", queueItem.namespaceKey))
		return err
	}

	switch queueItem.kind {
	case "CustomMetric":
		return h.handleCustomMetric(ns, name, queueItem)
	}

	return nil
}

func (h *Handler) handleCustomMetric(ns, name string, queueItem namespacedQueueItem) error {
	// check if item exists
	klog.V(2).Infof("processing item '%s' in namespace '%s'", name, ns)
	customMetricInfo, err := h.customMetricLister.CustomMetrics(ns).Get(name)

	if err != nil {
		if errors.IsNotFound(err) {
			// Then this we should remove
			klog.V(2).Infof("removing item from cache '%s' in namespace '%s'", name, ns)
			h.metriccache.Remove(queueItem.Key())
			return nil
		}
		return err
	}

	// TODO: Map the new fields here for Service Bus
	metric := custommetrics.MetricLastPointRequest{
		MetricName: customMetricInfo.ObjectMeta.Name,
		Counter:    customMetricInfo.Spec.MetricConfig.Counter,
		Endpoint:   customMetricInfo.Spec.MetricConfig.Endpoint,
	}

	klog.V(2).Infof("adding to cache item '%s' in namespace '%s'", name, ns)
	h.metriccache.Update(queueItem.Key(), name, metric)
	return nil
}
