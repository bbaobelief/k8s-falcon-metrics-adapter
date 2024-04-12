module github.com/bbaobelief/k8s-falcon-metrics-adapter

go 1.15

require (
	github.com/jsturtevant/gitsem v1.0.4
	github.com/kubernetes-sigs/custom-metrics-apiserver v0.0.0-20210311094424-0ca2b1909cdc
	github.com/patrickmn/go-cache v2.1.0+incompatible
	gopkg.in/blang/semver.v1 v1.1.0 // indirect
	k8s.io/api v0.20.5
	k8s.io/apimachinery v0.20.5
	k8s.io/client-go v0.20.5
	k8s.io/code-generator v0.20.5
	k8s.io/component-base v0.20.5
	k8s.io/klog/v2 v2.4.0
	k8s.io/metrics v0.20.5
)
