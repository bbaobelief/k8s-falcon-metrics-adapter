package controller

import (
	"fmt"
	"time"

	"github.com/bbaobelief/k8s-falcon-metrics-adapter/pkg/apis/metrics/v1alpha1"

	"k8s.io/apimachinery/pkg/util/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	informers "github.com/bbaobelief/k8s-falcon-metrics-adapter/pkg/client/informers/externalversions/metrics/v1alpha1"
)

// Controller will do the work of syncing the external metrics the metric adapter knows about.
type Controller struct {
	metricQueue        workqueue.RateLimitingInterface
	customMetricSynced cache.InformerSynced
	enqueuer           func(obj interface{})
	metricHandler      ControllerHandler
}

// NewController returns a new controller for handling external and custom metric types
func NewController(customMetricInformer informers.CustomMetricInformer, metricHandler ControllerHandler) *Controller {
	controller := &Controller{
		metricQueue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "metrics"),
		metricHandler:      metricHandler,
		customMetricSynced: customMetricInformer.Informer().HasSynced,
	}

	// wire up enque step.  This provides a hook for testing enqueue step
	controller.enqueuer = controller.enqueueExternalMetric

	klog.Info("Setting up custom metric event handlers")
	customMetricInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueuer,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueuer(new)
		},
		DeleteFunc: controller.enqueuer,
	})

	return controller
}

// Run is the main path of execution for the controller loop
func (c *Controller) Run(numberOfWorkers int, interval time.Duration, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.metricQueue.ShutDown()

	klog.V(2).Info("initializing controller")

	// do the initial synchronization (one time) to populate resources
	if !cache.WaitForCacheSync(stopCh, c.customMetricSynced) {
		runtime.HandleError(fmt.Errorf("Error syncing controller cache"))
		return
	}

	klog.V(2).Infof("starting %d workers with %d interval", numberOfWorkers, interval)
	for i := 0; i < numberOfWorkers; i++ {
		go wait.Until(c.runWorker, interval, stopCh)
	}

	<-stopCh
	klog.Info("Shutting down workers")
	return
}

func (c *Controller) runWorker() {
	klog.V(2).Info("Worker starting")

	for c.processNextItem() {
		klog.V(4).Info("processing next item")
	}

	klog.V(2).Info("worker completed")
}

func (c *Controller) processNextItem() bool {
	klog.V(4).Info("processing item")

	rawItem, quit := c.metricQueue.Get()
	if quit {
		klog.V(2).Info("recieved quit signal")
		return false
	}

	defer c.metricQueue.Done(rawItem)

	var queueItem namespacedQueueItem
	var ok bool
	if queueItem, ok = rawItem.(namespacedQueueItem); !ok {
		// not valid key do not put back on queue
		c.metricQueue.Forget(rawItem)
		runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", rawItem))
		return true
	}

	err := c.metricHandler.Process(queueItem)
	if err != nil {
		retrys := c.metricQueue.NumRequeues(rawItem)
		if retrys < 5 {
			klog.Errorf("Transient error with %d retrys for key %s: %s", retrys, rawItem, err)
			c.metricQueue.AddRateLimited(rawItem)
			return true
		}

		// something was wrong with the item on queue
		klog.Errorf("Max retries hit for key %s: %s", rawItem, err)
		c.metricQueue.Forget(rawItem)
		utilruntime.HandleError(err)
		return true
	}

	//if here success for get item
	klog.V(2).Infof("succesfully proccessed item '%s'", queueItem)
	c.metricQueue.Forget(rawItem)
	return true
}

func (c *Controller) enqueueExternalMetric(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}

	kind := getKind(obj)

	klog.V(2).Infof("adding item to queue for '%s' with kind '%s'", key, kind)
	c.metricQueue.AddRateLimited(namespacedQueueItem{
		namespaceKey: key,
		kind:         kind,
	})
}

type namespacedQueueItem struct {
	namespaceKey string
	kind         string
}

func (q namespacedQueueItem) Key() string {
	return fmt.Sprintf("%s/%s", q.kind, q.namespaceKey)
}

func getKind(obj interface{}) string {
	// Due to this issue https://github.com/kubernetes/apiextensions-apiserver/issues/29
	// metadata is not set on freshly set CRD's
	// So the following does not work:
	// 		t, err := meta.TypeAccessor(obj)
	// 		kind := t.GetKind() // Kind will be blank
	//
	// A possible alternative to switching on type would be to use
	// 		https://github.com/kubernetes/kubernetes/blob/7f23a743e8c23ac6489340bbb34fa6f1d392db9d/pkg/kubectl/cmd/util/conversion.go
	// 		v := cmdUtil.AsDefaultVersionedOrOriginal(item, nil)
	// 		k := v.GetObjectKind().GroupVersionKind().Kind
	//
	// Instead use type to predict Kind which is good enough for our purposes:

	switch obj.(type) {
	case *v1alpha1.CustomMetric:
		return "CustomMetric"
	default:
		klog.Error("No known type of object")
		return ""
	}
}
