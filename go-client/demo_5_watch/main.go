package main

import (
	"context"
	"flag"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
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
	timeout := int64(50)
	watcher, err := clientSet.CoreV1().Pods("default").Watch(context.TODO(), metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		fmt.Printf("err %s", err.Error())
	}
	for event := range watcher.ResultChan() {
		item := event.Object.(*corev1.Pod)

		switch event.Type {
		case watch.Added:
			fmt.Printf("Pod Added: %s\n", item.GetName())
			processPod(item.GetName())
		case watch.Modified:
			fmt.Printf("Pod Modified: %s\n", item.GetName())
		case watch.Deleted:
			fmt.Printf("Pod Deleted: %s\n", item.GetName())
		}
	}
}

func processPod(podName string) {
	fmt.Printf("Processing Pod: %s\n", podName)
}
