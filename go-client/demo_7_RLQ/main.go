package main

import (
	"flag"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "/root/.kube/config", "Path to the kubeconfig file")
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		fmt.Printf("err %s", err.Error())
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("err %s", err.Error())
	}

	// 初始化 informer
	informerFactory := informers.NewSharedInformerFactory(clientSet, time.Hour*12)

	// 创建一个 RLQ 队列
	queue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[string]())

	// 对 deployment 进行监听
	deployInformer := informerFactory.Apps().V1().Deployments()
	informer := deployInformer.Informer()
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    func(obj interface{}) { onAddDeployment(obj, queue) },
			UpdateFunc: func(oldObj, newObj interface{}) { onUpdateDeployment(oldObj, newObj, queue) },
			DeleteFunc: func(obj interface{}) { onDeleteDeployment(obj, queue) },
		},
	)

	controller := NewController(queue, deployInformer.Informer().GetIndexer(), informer)
	stopper := make(chan struct{})
	defer close(stopper)

	// 启动 informer
	informerFactory.Start(stopper)
	informerFactory.WaitForCacheSync(stopper)

	// 处理队列事件
	go func() {
		for {
			if !controller.processNextItem() {
				break
			}
		}
	}()
	<-stopper
}

type controller struct {
	indexer  cache.Indexer
	queue    workqueue.TypedRateLimitingInterface[string]
	informer cache.Controller
}

func NewController(queue workqueue.TypedRateLimitingInterface[string], indexer cache.Indexer, informer cache.Controller) *controller {
	return &controller{
		indexer:  indexer,
		queue:    queue,
		informer: informer,
	}
}

func (c *controller) processNextItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	err := c.syncToStdout(key)
	c.handleErr(err, key)
	return true
}

func (c *controller) syncToStdout(key string) error {
	// 通过 key 直接从 indexer 中获取对象
	Obj, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		fmt.Printf("Fetching object with key %s from store failed with %v\n", key, err)
		return err
	}

	if !exists {
		fmt.Printf("Pod %s does not exist anymore\n", key)
	} else {
		deployment := Obj.(*appsv1.Deployment)
		fmt.Printf("Sync/Add/Update for Deployment %s\n", deployment.GetName())
		if deployment.Name == "test-deployment" {
			time.Sleep(2 * time.Second)
			return fmt.Errorf("test-deployment is not allowed")
		}
	}
	return nil
}

func (c *controller) handleErr(err error, key string) {
	if err == nil {
		c.queue.Forget(key)
		return
	}
	if c.queue.NumRequeues(key) < 5 {
		fmt.Printf("Retry %d for key %s\n", c.queue.NumRequeues(key), key)
		c.queue.AddRateLimited(key)
		return
	}

	c.queue.Forget(key)
	fmt.Printf("Dropping pod %v out of the queue: %v\n", key, err)
}

func onAddDeployment(obj interface{}, queue workqueue.TypedRateLimitingInterface[string]) {
	key, err := cache.MetaNamespaceKeyFunc(obj) //namespace/name
	if err == nil {
		queue.Add(key)
	}
}

func onUpdateDeployment(oldObj, newObj interface{}, queue workqueue.TypedRateLimitingInterface[string]) {
	key, err := cache.MetaNamespaceKeyFunc(newObj) //namespace/name
	if err == nil {
		queue.Add(key)
	}
}

func onDeleteDeployment(obj interface{}, queue workqueue.TypedRateLimitingInterface[string]) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj) //namespace/name
	if err == nil {
		queue.Add(key)

	}
}
