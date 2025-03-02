package main

import (
	"context"
	"flag"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "/root/.kube/config", "Path to the kubeconfig file")
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		fmt.Printf("err %s", err.Error())
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		fmt.Printf("err %s", err.Error())
	}

	// 指定 GVR
	gvr := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "pods",
	}

	// 获取 Pod 列表
	unStructPodList, err := dynamicClient.Resource(gvr).Namespace("kube-system").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("err %s", err.Error())
	}

	// 定义一个 PodList Struct
	podList := &corev1.PodList{}
	// 将 unstructured 对象转换为 PodList
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unStructPodList.UnstructuredContent(), podList)
	if err != nil {
		fmt.Printf("err %s", err.Error())
	}
	for _, pod := range podList.Items {
		fmt.Printf("Pod Name: %s, %s, %s\n", pod.Name, pod.Namespace, pod.Status.Phase)
	}
}
