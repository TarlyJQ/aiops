package main

import (
	"context"
	"flag"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "/root/.kube/config", "Path to the kubeconfig file")
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		fmt.Printf("err %s", err.Error())
	}
	config.APIPath = "api"
	config.GroupVersion = &corev1.SchemeGroupVersion
	config.NegotiatedSerializer = scheme.Codecs

	// 初始化 restclient
	restClint, err := rest.RESTClientFor(config)
	if err != nil {
		fmt.Printf("err %s", err.Error())
	}

	// 创建一个空结构体，存储 pod 列表
	podList := &corev1.PodList{}
	restClint.Get().Namespace("kube-system").Resource("pods").Do(context.TODO()).Into(podList)
	for _, pod := range podList.Items {
		fmt.Printf("pod name %s, %s, %s\n", pod.Name, pod.Namespace, pod.Status.Phase)
	}
}
