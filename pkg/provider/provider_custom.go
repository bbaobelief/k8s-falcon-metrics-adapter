// Package provider is the implementation of custom metric and external metric apis
// see https://github.com/kubernetes/community/blob/master/contributors/design-proposals/instrumentation/custom-metrics-api.md#api-paths
package provider

import (
	"fmt"
	"strings"
	"time"

	"github.com/bbaobelief/k8s-falcon-metrics-adapter/pkg/exec"
	"github.com/bbaobelief/k8s-falcon-metrics-adapter/pkg/falcon/custommetrics"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/custom_metrics"

	"github.com/kubernetes-sigs/custom-metrics-apiserver/pkg/provider"
	"github.com/kubernetes-sigs/custom-metrics-apiserver/pkg/provider/helpers"
)

// GetMetricByName fetches a particular metric for a particular object.
// The namespace will be empty if the metric is root-scoped.
func (p *FalconProvider) GetMetricByName(name types.NamespacedName, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValue, error) {
	// not implemented yet
	return nil, errors.NewServiceUnavailable("not implemented yet")
}

// GetMetricBySelector fetches a particular metric for a set of objects matching
// the given label selector.  The namespace will be empty if the metric is root-scoped.
func (p *FalconProvider) GetMetricBySelector(namespace string, selector labels.Selector, metricInfo provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValueList, error) {
	klog.V(0).Infof("Received request for custom_metric: group_resource: %s, namespace: %s, metric_name: %s, selectors: %s", metricInfo.GroupResource.String(), namespace, metricInfo.Metric, selector.String())

	_, selectable := selector.Requirements()
	if !selectable {
		return nil, errors.NewBadRequest("label is set to not selectable. this should not happen")
	}

	metricRequestInfo := p.getCustomMetricRequest(namespace, selector, metricInfo)

	resourceNames, err := helpers.ListObjectNames(p.mapper, p.kubeClient, namespace, selector, metricInfo)
	if err != nil {
		klog.Errorf("not able to list objects from api server: %v", err)
		return nil, errors.NewInternalError(fmt.Errorf("not able to list objects from api server for this resource"))
	}

	// TODO: Add support for app insights where pods are mapped 1 to 1.
	metricList := make([]custom_metrics.MetricValue, 0)
	for _, podName := range resourceNames {
		ref, err := helpers.ReferenceFor(p.mapper, types.NamespacedName{Namespace: namespace, Name: podName}, metricInfo)
		if err != nil {
			return nil, err
		}

		// TODO pod --> hostname or dns?
		containerName := PodNameToContainerName(podName)
		ex := exec.NewExecWithOptions(namespace, podName, containerName)
		hostname, stderr, err := ex.ExecCommandInPod(p.clientSet, p.kubeConfig)
		if err != nil {
			klog.Errorf("NewExecWithOptions, hostname=%s stderr=%s err=%v", hostname, stderr, err)
		}

		metricRequestInfo.Endpoint = hostname

		// query falcon api
		val, err := p.falconClient.GetCustomMetric(metricRequestInfo)
		if err != nil {
			klog.Fatalf("bad request: %v", err)
			return nil, errors.NewBadRequest(err.Error())
		}

		metricValue := custom_metrics.MetricValue{
			DescribedObject: ref,
			Metric: custom_metrics.MetricIdentifier{
				Name: metricInfo.Metric,
			},
			Timestamp: metav1.Time{time.Now()},
			Value:     *resource.NewMilliQuantity(int64(val*1000), resource.DecimalSI),
		}

		// add back the meta data about the request selectors
		if len(selector.String()) > 0 {
			labelSelector, err := metav1.ParseToLabelSelector(selector.String())
			if err != nil {
				return nil, err
			}
			metricValue.Metric.Selector = labelSelector
		}

		metricList = append(metricList, metricValue)
	}

	return &custom_metrics.MetricValueList{
		Items: metricList,
	}, nil
}

// ListAllMetrics provides a list of all available metrics at
// the current time.  Note that this is not allowed to return
// an error, so it is reccomended that implementors cache and
// periodically update this list, instead of querying every time.
func (p *FalconProvider) ListAllMetrics() []provider.CustomMetricInfo {
	p.valuesLock.RLock()
	defer p.valuesLock.RUnlock()

	var customMetricsInfo []provider.CustomMetricInfo
	for _, name := range p.metricCache.ListMetricNames() {
		// only process if name is non-empty
		if name != "" {
			info := provider.CustomMetricInfo{
				GroupResource: schema.GroupResource{Group: "", Resource: "pods"},
				Metric:        name,
				Namespaced:    true,
			}
			customMetricsInfo = append(customMetricsInfo, info)
		}
	}
	return customMetricsInfo
}

func (p *FalconProvider) getCustomMetricRequest(namespace string, selector labels.Selector, info provider.CustomMetricInfo) custommetrics.MetricLastPointRequest {
	// todo get request cache
	cachedRequest, found := p.metricCache.GetCustomRequest(namespace, info.Metric)
	if found {
		return cachedRequest
	}

	// TODO metric Objects ---> counter
	// because counter are multipart in falcon and we can not pass an extra /
	// through k8s api we convert - to / to get around that
	convertedMetricName := strings.Replace(info.Metric, "-equal-", "=", -1)
	convertedMetricName = strings.Replace(convertedMetricName, "-", "/", -1)
	klog.V(2).Infof("New call to GetCustomMetric, CustomMetric: %s counter: %s", info.Metric, convertedMetricName)
	metricRequestInfo := custommetrics.NewFalconMetricRequest(info.Metric, convertedMetricName)

	return metricRequestInfo
}
