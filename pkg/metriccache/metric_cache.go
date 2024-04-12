package metriccache

import (
	"fmt"
	"sync"

	"github.com/bbaobelief/k8s-falcon-metrics-adapter/pkg/falcon/custommetrics"
	"k8s.io/klog/v2"
)

// MetricCache holds the loaded metric request info in the system
type MetricCache struct {
	metricMutext   sync.RWMutex
	metricRequests map[string]interface{}
	metricNames    map[string]string
}

// NewMetricCache creates the cache
func NewMetricCache() *MetricCache {
	return &MetricCache{
		metricRequests: make(map[string]interface{}),
		metricNames:    make(map[string]string),
	}
}

// Update sets a metric request in the cache
func (mc *MetricCache) Update(key, name string, metricRequest interface{}) {
	mc.metricMutext.Lock()
	defer mc.metricMutext.Unlock()

	mc.metricRequests[key] = metricRequest
	mc.metricNames[key] = name
}

// GetCustomRequest retrieves a metric request from the cache
func (mc *MetricCache) GetCustomRequest(namespace, name string) (custommetrics.MetricLastPointRequest, bool) {
	mc.metricMutext.RLock()
	defer mc.metricMutext.RUnlock()

	key := customMetricKey(namespace, name)
	metricRequest, exists := mc.metricRequests[key]
	if !exists {
		klog.V(2).Infof("INFO: GetCustomRequest, request not found %s", key)
		return custommetrics.MetricLastPointRequest{}, false
	}
	klog.V(2).Infof("INFO: GetCustomRequest, found request cache: %s", key)
	return metricRequest.(custommetrics.MetricLastPointRequest), true
}

// Remove retrieves a metric request from the cache
func (mc *MetricCache) Remove(key string) {
	mc.metricMutext.Lock()
	defer mc.metricMutext.Unlock()

	delete(mc.metricRequests, key)
	delete(mc.metricNames, key)
}

// ListMetricNames retrieves a list of metric names from the cache.
func (mc *MetricCache) ListMetricNames() []string {
	//klog.Info("ListMetricNames ", mc.metricNames)
	keys := make([]string, len(mc.metricNames))
	for k := range mc.metricNames {
		keys = append(keys, mc.metricNames[k])
	}

	return keys
}

func customMetricKey(namespace string, name string) string {
	return fmt.Sprintf("CustomMetric/%s/%s", namespace, name)
}
