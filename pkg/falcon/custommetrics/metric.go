package custommetrics

import "fmt"

type MetricLastPointRequest struct {
	MetricName string `json:"-"`

	Endpoint string `json:"endpoint"`
	Counter  string `json:"counter"`
}

// NewMetricRequest creates a new metric request with defaults for optional parameters
func NewFalconMetricRequest(metricName, counter string) MetricLastPointRequest {
	return MetricLastPointRequest{
		MetricName: metricName,
		Counter:    counter,
	}
}

// MetricsResult a metric result.
type MetricsResult []struct {
	Endpoint string `json:"endpoint"`
	Counter  string `json:"counter"`
	Values   struct {
		Timestamp int64   `json:"timestamp"`
		Value     float64 `json:"value"`
	} `json:"value"`
}

func CustomMetricResultKey(endpoint string, counter string) string {
	return fmt.Sprintf("customMetricResult/%s/%s", endpoint, counter)
}
