### 描述
k8s-falcon-metrics-adapter 实现了HPA自定义指标，可以从falcon获取任意监控指标配置多维度HPA伸缩。

### 部署
```
make all
kubectl apply -f deploy/adapter.yaml
```

### 示例
- 创建Deployment
```
➜ kubectl apply -f samples/deployment.yaml 
namespace/custom-metrics-test created
deployment.apps/test-sample created
service/test-sample created
```
- 创建custom metric
```$xslt
➜ kubectl apply -f samples/custommetric.yaml
custommetric.metrics.falcon/cpu.busy created
custommetric.metrics.falcon/net.if.in.bytes-iface-equal-eth0 created
```
- 创建自定义指标hpa
```$xslt
➜ kubectl apply -f samples/hpa-custom.yaml  
horizontalpodautoscaler.autoscaling/test-sample created
```
- 查看所有自定义指标
```$xslt
# kubectl get fcm -A
NAMESPACE             NAME                               AGE
custom-metrics-test   cpu.busy                           104m
custom-metrics-test   net.if.in.bytes-iface-equal-eth0   104m

# kubectl get --raw "/apis/custom.metrics.k8s.io/v1beta1"|jq
{
  "kind": "APIResourceList",
  "apiVersion": "v1",
  "groupVersion": "custom.metrics.k8s.io/v1beta1",
  "resources": [
    {
      "name": "pods/cpu.busy",
      "singularName": "",
      "namespaced": true,
      "kind": "MetricValueList",
      "verbs": [
        "get"
      ]
    },
    {
      "name": "pods/net.if.in.bytes-iface-equal-eth0",
      "singularName": "",
      "namespaced": true,
      "kind": "MetricValueList",
      "verbs": [
        "get"
      ]
    }
  ]
}
```
- 查看单个指标
```$xslt
# kubectl get --raw "/apis/custom.metrics.k8s.io/v1beta1/namespaces/custom-metrics-test/pods/*/cpu.busy"|jq
{
  "kind": "MetricValueList",
  "apiVersion": "custom.metrics.k8s.io/v1beta1",
  "metadata": {
    "selfLink": "/apis/custom.metrics.k8s.io/v1beta1/namespaces/custom-metrics-test/pods/%2A/cpu.busy"
  },
  "items": [
    {
      "describedObject": {
        "kind": "Pod",
        "namespace": "custom-metrics-test",
        "name": "test-sample-59f6b8979b-dcsdn",
        "apiVersion": "/v1"
      },
      "metricName": "cpu.busy",
      "timestamp": "2021-04-08T07:15:48Z",
      "value": "0",
      "selector": null
    },
    {
      "describedObject": {
        "kind": "Pod",
        "namespace": "custom-metrics-test",
        "name": "test-sample-59f6b8979b-r498v",
        "apiVersion": "/v1"
      },
      "metricName": "cpu.busy",
      "timestamp": "2021-04-08T07:15:48Z",
      "value": "0",
      "selector": null
    }
  ]
}
```
- 查看指标详细信息
```$xslt
# kubectl get --raw "/apis/metrics.falcon/v1alpha1/custommetrics"|jq
# kubectl get --raw "/apis/metrics.falcon/v1alpha1/namespaces/custom-metrics-test/custommetrics/cpu.busy"|jq
```
- watch hpa状态
```$xslt
# kubectl get hpa -A -w
NAMESPACE             NAME          REFERENCE                TARGETS            MINPODS   MAXPODS   REPLICAS   AGE
custom-metrics-test   test-sample   Deployment/test-sample   0/50, 0/52428800   2         5         2          9m7s
```
- 模拟测试数据
```$xslt
python hack/fake_falcon_push.py
```

- 观察扩容event
```$xslt
# kubectl get hpa -A -w
NAMESPACE             NAME          REFERENCE                TARGETS                       MINPODS   MAXPODS   REPLICAS   AGE
custom-metrics-test   test-sample   Deployment/test-sample   43/50, 1051660500m/52428800   2         5         2          102m
custom-metrics-test   test-sample   Deployment/test-sample   38/50, 1052783500m/52428800   2         5         2          102m
custom-metrics-test   test-sample   Deployment/test-sample   55/50, 1051657500m/52428800   2         5         2          103m


# kubectl describe deployments test-sample -n custom-metrics-test
Name:                   test-sample
Namespace:              custom-metrics-test
CreationTimestamp:      Thu, 08 Apr 2021 15:08:25 +0800
Labels:                 <none>
Annotations:            deployment.kubernetes.io/revision: 1
Selector:               app=test-sample
Replicas:               3 desired | 3 updated | 3 total | 3 available | 0 unavailable
StrategyType:           RollingUpdate
MinReadySeconds:        0
RollingUpdateStrategy:  25% max unavailable, 25% max surge
Pod Template:
  Labels:  app=test-sample
  Containers:
   test-sample:
    Image:      jsturtevant/metric-rps-example
    Port:       <none>
    Host Port:  <none>
    Command:
      top
      -b
    Environment:  <none>
    Mounts:       <none>
  Volumes:        <none>
Conditions:
  Type           Status  Reason
  ----           ------  ------
  Progressing    True    NewReplicaSetAvailable
  Available      True    MinimumReplicasAvailable
OldReplicaSets:  <none>
NewReplicaSet:   test-sample-59f6b8979b (3/3 replicas created)
Events:
  Type    Reason             Age   From                   Message
  ----    ------             ----  ----                   -------
  Normal  ScalingReplicaSet  88s   deployment-controller  Scaled up replica set test-sample-59f6b8979b to 3
```


### todo
- [x] code-generator生成client
- [x] client-go注册使用clientset,informers,listers
- [x] ListAllMetrics缓存更新策略,过期metric清理
- [x] GetMetricBySelector获取podname
- [x] RESTClient注册,podname通过exec获取hostname
- [x] k8s custom metric转换falcon counter
- [x] falcon api调用,timeout设置
- [x] 超过5分钟未更新的指标返回0
- [x] k8s hpa默认30s去falcon获取一次数据，但falcon指标60s更新一次。为了减少重复请求缓存value 50s
- [x] 部署crd,auth等k8s yaml编写
- [x] 部署示例yaml编写
- [x] 发布及编译脚本编写
- [ ] 测试用例

### 参考
- https://github.com/kubernetes-sigs/custom-metrics-apiserver
- https://github.com/awslabs/k8s-cloudwatch-adapter