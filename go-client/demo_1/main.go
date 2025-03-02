package main

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	config, err := rest.InClusterConfig()
	// kubeconfig := flag.String("kubeconfig", "/root/.kube/config", "Path to the kubeconfig file")
	// config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		fmt.Printf("err %s", err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("err %s", err.Error())
	}
	//pod
	pods, err := clientset.CoreV1().Pods("helm-app").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("err %s", err.Error())
	}
	for _, pod := range pods.Items {
		fmt.Println(pod.Name)
	}
	// deployment
	deployments, err := clientset.AppsV1().Deployments("helm-app").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("err %s", err.Error())
	}
	for _, deployment := range deployments.Items {
		fmt.Println(deployment.Name)
	}
}
