package main

import (
	"context"
	"flag"
	"fmt"
	"strings"
	_ "embed"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

//go:embed deployment.yaml
var deploymentYaml string

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

	// 解析 yaml 文件 转换成 unstructured
	deployObj := &unstructured.Unstructured{}
	if err := yaml.Unmarshal([]byte(deploymentYaml), deployObj); err != nil {
		fmt.Printf("err %s", err.Error())
	}

	// 从 deployOjb 获取 GVK
	apiVersion, found, err := unstructured.NestedString(deployObj.Object, "apiVersion")
	if err != nil || !found {
		fmt.Printf("err %s", err.Error())
	}

	kind, found, err := unstructured.NestedString(deployObj.Object, "kind")
	if err != nil || !found {
		fmt.Printf("err %s", err.Error())
	}

	// 转化 GVR
	gvr := schema.GroupVersionResource{}
	versionParts := strings.Split(apiVersion, "/")
	if len(versionParts) == 2 {
		gvr.Group = versionParts[0]
		gvr.Version = versionParts[1]
	} else {
		gvr.Version = versionParts[0]
	}

	switch kind {
	case "Deployment":
		gvr.Resource = "deployments"
	case "Service":
		gvr.Resource = "services"
	case "Pod":
		gvr.Resource = "pods"
	default:
		fmt.Printf("kind %s not supported", kind)
	}

	// 使用 dynamicCLient 创建资源
	_, err = dynamicClient.Resource(gvr).Namespace("default").Create(context.TODO(), deployObj, v1.CreateOptions{})
	if err != nil {
		fmt.Printf("err %s", err.Error())
	}
	fmt.Print("Create resource successfully")
}
