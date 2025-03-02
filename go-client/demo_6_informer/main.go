package main

import (
	"flag"
	"fmt"
	"time"

	// corev1 "k8s.io/api/core/v1"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
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

	//Deployment
	deploymentInformer := informerFactory.Apps().V1().Deployments()
	informer := deploymentInformer.Informer()
	deployLister := deploymentInformer.Lister()
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				fmt.Printf("Deployment added \n")
			},

			DeleteFunc: func(obj interface{}) {
				fmt.Printf("Deployment deleted \n")
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				fmt.Printf("Deployment updated \n")
			},
		},
	)

	// service
	serviceInformer := informerFactory.Core().V1().Services()
	informer = serviceInformer.Informer()
	// serviceLister := serviceInformer.Lister()
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				fmt.Printf("Service added \n")
			},

			DeleteFunc: func(obj interface{}) {
				fmt.Printf("Service deleted \n")
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				fmt.Printf("Service updated \n")
			},
		},
	)

	stopper := make(chan struct{})
	defer close(stopper)

	// 启动 informer，List && Watch
	informerFactory.Start(stopper)
	// 等待所有 informer 同步完成
	informerFactory.WaitForCacheSync(stopper)

	deployments, err := deployLister.Deployments("default").List(labels.Everything())
	if err != nil {
		fmt.Printf("err %s", err.Error())
	}
	for idx, deploy := range deployments {
		fmt.Println(idx, deploy.Name)
	}
	<-stopper
}
