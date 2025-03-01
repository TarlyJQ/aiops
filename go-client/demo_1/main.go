package main

import (
	"context"
	"flag"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "/root/.kube/config", "Path to the kubeconfig file")
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		fmt.Println(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println(err)
	}
	pods, _ := clientset.CoreV1().Pods("helm-app").List(context.TODO(), metav1.ListOptions{})
	for _, pod := range pods.Items {
		fmt.Println(pod.Name)
	}
	// deployment
	deployments, _ := clientset.AppsV1().Deployments("helm-app").List(context.TODO(), metav1.ListOptions{})
	for _, deployment := range deployments.Items {
		fmt.Println(deployment.Name)
	}
}
