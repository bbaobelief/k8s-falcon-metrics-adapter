package custommetrics

import (
	"k8s.io/klog/v2"
)

func normalizeValue(value interface{}) float64 {
	switch t := value.(type) {
	case int32:
		return float64(value.(int32))
	case float32:
		return float64(value.(float32))
	case float64:
		return value.(float64)
	case int64:
		return float64(value.(int64))
	default:
		klog.V(0).Infof("unexpected type: %T", t)
		return 0
	}
}
