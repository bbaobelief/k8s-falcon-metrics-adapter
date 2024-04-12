package custommetrics

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/patrickmn/go-cache"
	"io"
	"io/ioutil"
	"k8s.io/klog/v2"
	"net/http"
	"os"
	"time"
)

const (
	defaultAPIUrl  = "api.monitor.ik.srv"
	defaultApiSig  = "default-token-used-in-server-side"
	defaultApiName = "k8s-falcon-metrics-adapter"
	defaultTimeout = 10 * time.Second
)

type FalconClient interface {
	GetCustomMetric(request MetricLastPointRequest) (float64, error)
}

// falconClient is used to call falcon Api
type falconClient struct {
	url  string
	name string
	sig  string

	cache *cache.Cache
}

// NewClient creates a client for falcon api
func NewClient() FalconClient {
	falconApiUrl := os.Getenv("FALCON_API_URL")
	falconApiName := os.Getenv("FALCON_API_NAME")
	falconApiSig := os.Getenv("FALCON_API_SIG")

	if falconApiSig == "" {
		falconApiName = defaultApiName
		falconApiSig = defaultApiSig
	}

	if falconApiUrl == "" {
		falconApiUrl = defaultAPIUrl
	}

	return falconClient{
		url:   falconApiUrl,
		name:  falconApiName,
		sig:   falconApiSig,
		cache: cache.New(50*time.Second, 10*time.Minute),
	}
}

// GetCustomMetric calls to falcon api to retrieve the value of the metric requested
func (c falconClient) GetCustomMetric(request MetricLastPointRequest) (float64, error) {
	key := CustomMetricResultKey(request.Endpoint, request.Counter)

	// TODO add cache
	if x, found := c.cache.Get(key); found {
		val := x.(float64)
		klog.V(2).Infof("GetCustomMetric, found cache, key: %s value: %v", key, val)
		return val, nil
	}

	lastPointRequest := []MetricLastPointRequest{}
	lastPointRequest = append(lastPointRequest, request)

	metricsResult, err := getMetricLastPoint(c, lastPointRequest)
	if err != nil {
		return 0, err
	}

	if metricsResult == nil {
		return 0, errors.New("GetCustomMetric, metrics result is nil.")
	}

	if len(*metricsResult) <= 0 {
		klog.Errorf("GetCustomMetric, metrics result len=0.")
		return 0, nil
	}

	metric := (*metricsResult)[0]

	klog.V(3).Infof("GetCustomMetric, request: %s value: %v", request, metric.Values.Value)

	// The time interval cannot be greater than 5 minutes
	interval := time.Now().Unix() - metric.Values.Timestamp
	if interval >= 300 {
		klog.Errorf("GetCustomMetric, interval: %ds > 300s endpoint: %s counter: %s timestamp: %d value: %v", interval, request.Endpoint, request.Counter, metric.Values.Timestamp, metric.Values.Value)
		return 0, nil
	}

	value := normalizeValue(metric.Values.Value)

	// update cache
	c.cache.Set(key, value, cache.DefaultExpiration)
	klog.V(2).Infof("GetCustomMetric, set cache, key: %s value: %v", key, value)

	return value, nil
}

// GetMetric calls to API to retrieve a specific metric
func getMetricLastPoint(c falconClient, metricInfo []MetricLastPointRequest) (*MetricsResult, error) {
	client := &http.Client{Timeout: defaultTimeout}

	token, _ := json.Marshal(map[string]string{
		"name": c.name,
		"sig":  c.sig,
	})

	payload, _ := json.Marshal(metricInfo)
	req, _ := http.NewRequest("POST", fmt.Sprintf("http://%s/api/v1/graph/lastpoint", c.url), bytes.NewBuffer(payload))

	req.Header.Add("Apitoken", string(token))
	req.Header.Add("content-type", "application/json")

	klog.V(4).Infof("INFO: getMetricLastPoint, request url: %s payload: %s", req.URL, payload)
	resp, err := client.Do(req)
	if err != nil {
		klog.Errorf("unable to retrive metric: %v", err)
		return nil, err
	}

	// check the response status is OK. If not, return the error
	if resp.StatusCode != 200 {
		defer resp.Body.Close()
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			klog.Errorf("unable to retrieve metric: %s", err)
			return nil, err
		}

		respMessage := string(respBody)
		err = fmt.Errorf(respMessage)
		return nil, err
	}
	metricsResult := MetricsResult{}
	return unmarshalResponse(resp.Body, &metricsResult)
}

func unmarshalResponse(body io.ReadCloser, metricsResult *MetricsResult) (*MetricsResult, error) {
	defer body.Close()
	respBody, err := ioutil.ReadAll(body)

	if err != nil {
		klog.Errorf("unable to get read metric response body: %v", err)
		return nil, err
	}

	err = json.Unmarshal(respBody, metricsResult)
	if err != nil {
		return nil, errors.New("unknown response format")
	}

	return metricsResult, nil
}
