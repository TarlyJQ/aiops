/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/TarlyJQ/aiops/k8scopilot/cmd/utils"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/restmapper"
)

// deepseekCmd represents the deepseek command
var deepseekCmd = &cobra.Command{
	Use:   "deepseek",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		startChat()
	},
}

func startChat() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("我是 K8s Copilot， 请问有什么可以帮助你？")
	for {
		fmt.Print("> ")
		if scanner.Scan() {
			input := scanner.Text()
			if input == "exit" {
				fmt.Println("再见！")
				break
			}
			if input == "" {
				continue
			}
			response := processInput(input)
			fmt.Println(response)
		}
	}
}

func processInput(input string) string {
	client, err := utils.NewOpenAIClient()
	if err != nil {
		return err.Error()
	}
	response := functionCalling(input, client)
	return response
}

// func sanitizeYAML(raw string) string {
// 	// 强化版正则表达式（处理所有边界情况）
// 	re := regexp.MustCompile(
// 		`(?m)(^|\n)` + // 匹配开头或换行
// 			`\x60{3}(yaml)?` + // 匹配 ``` 或 ```yaml
// 			`(\n|$)` + // 匹配后续换行或结尾
// 			`|(\n?)\x60{3}$`, // 专门处理结尾的 ```
// 	)

// 	// 两阶段清理
// 	step1 := re.ReplaceAllString(raw, "")

// 	// 额外处理可能的残留
// 	return strings.TrimSpace(
// 		strings.TrimPrefix(
// 			strings.TrimSuffix(step1, "`"),
// 			"`",
// 		),
// 	)
// }

func functionCalling(input string, client *utils.OpenAI) string {
	// 定义第一个函数 ，生成 K8s YAML 并且部署资源用的
	f1 := openai.FunctionDefinition{
		Name:        "generateAndDeployResource",
		Description: "生成 Kubernetes YAML 文件",
		Parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"user_input": {
					Type:        jsonschema.String,
					Description: "用户输入的原始内容",
				},
			},
			Required: []string{"user_input"},
		},
	}
	t1 := openai.Tool{
		Type:     openai.ToolTypeFunction,
		Function: &f1,
	}
	// 定义查询 K8s 资源
	f2 := openai.FunctionDefinition{
		Name:        "queryResource",
		Description: "查询 Kubernetes 资源",
		Parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"namespace": {
					Type:        jsonschema.String,
					Description: "Kubernetes 命名空间",
				},
				"resource_type": {
					Type:        jsonschema.String,
					Description: "Kubernetes 标准资源类型，例如 pod、deployment、service等",
				},
			},
			Required: []string{"namespace", "resource_type"},
		},
	}
	t2 := openai.Tool{
		Type:     openai.ToolTypeFunction,
		Function: &f2,
	}
	// 用来删除 K8s 资源
	f3 := openai.FunctionDefinition{
		Name:        "deleteResource",
		Description: "删除 Kubernetes 资源",
		Parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"namespace": {
					Type:        jsonschema.String,
					Description: "Kubernetes 命名空间",
				},
				"resource_type": {
					Type:        jsonschema.String,
					Description: "Kubernetes 标准资源类型，例如 pod、deployment、service等",
				},
				"resource_name": {
					Type:        jsonschema.String,
					Description: "Kubernetes 资源名称",
				},
			},
			Required: []string{"namespace", "resource_type", "resource_name"},
		},
	}
	t3 := openai.Tool{
		Type:     openai.ToolTypeFunction,
		Function: &f3,
	}
	// 调用 t1、t2、t3
	dialogue := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleUser, Content: input},
	}
	resp, err := client.Client.CreateChatCompletion(context.TODO(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT4o,
			Messages: dialogue,
			Tools:    []openai.Tool{t1, t2, t3},
		},
	)
	if err != nil {
		return err.Error()
	}

	msg := resp.Choices[0].Message
	if len(msg.ToolCalls) != 1 {
		return fmt.Sprintf("未找到合适的工具调用， %v", len(msg.ToolCalls))
	}
	// 组装对话历史
	dialogue = append(dialogue, msg)
	for _, msg := range dialogue {
		fmt.Printf("Role: %s, Content: %s\n", msg.Role, msg.Content)
	}
	// fmt.Sprintf("OpenAI 希望能请求函数 %s，参数 %s", msg.ToolCalls[0].Function.Name, msg.ToolCalls[0].Function.Arguments)
	result, err := callFunction(client, msg.ToolCalls[0].Function.Name, msg.ToolCalls[0].Function.Arguments)
	if err != nil {
		return err.Error()
	}
	return result
}

func callFunction(client *utils.OpenAI, name, arguments string) (string, error) {
	if name == "generateAndDeployResource" {
		params := struct {
			UserInput string `json:"user_input"`
		}{}
		if err := json.Unmarshal([]byte(arguments), &params); err != nil {
			return "", err
		}
		return generateAndDeployResource(client, params.UserInput)
	}
	if name == "queryResource" {
		params := struct {
			Namespace    string `json:"namespace"`
			ResourceType string `json:"resource_type"`
		}{}
		if err := json.Unmarshal([]byte(arguments), &params); err != nil {
			return "", err
		}
		return queryResource(params.Namespace, params.ResourceType)
	}
	if name == "deleteResource" {
		params := struct {
			Namespace    string `json:"namespace"`
			ResourceType string `json:"resource_type"`
			ResourceName string `json:"resource_name"`
		}{}
		if err := json.Unmarshal([]byte(arguments), &params); err != nil {
			return "", err
		}
		return deleteResource(params.Namespace, params.ResourceType, params.ResourceName)
	}
	return "", fmt.Errorf("未找到函数 %s", name)
}

func generateAndDeployResource(client *utils.OpenAI, userInput string) (string, error) {
	yamlContent, err := client.SendMessage("你现在是一个K8s 资源生成器，请根据用户的输入生成 K8s YAML， 注意除了 YAML 内容以外不要输出任务内容，不要把YAML内容放在```代码块中", userInput)
	if err != nil {
		return "", err
	}
	// return yamlContent, nil
	// TODO: 调用 dynamic client 部署资源
	clientGo, err := utils.NewClientGo(kubeconfig)
	if err != nil {
		return "", err
	}
	resources, err := restmapper.GetAPIGroupResources(clientGo.DiscoveryClient)
	if err != nil {
		return "", err
	}
	// 把 YAML 转化成 Unstructured
	unstructuredObj := &unstructured.Unstructured{}
	_, _, err = scheme.Codecs.UniversalDeserializer().Decode([]byte(yamlContent), nil, unstructuredObj)
	if err != nil {
		return "", err
	}

	// 创建 mapper
	mapper := restmapper.NewDiscoveryRESTMapper(resources)

	// 获取 GVK
	gvk := unstructuredObj.GroupVersionKind()
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return "", err
	}
	namespace := unstructuredObj.GetNamespace()
	if namespace == "" {
		namespace = "default"
	}
	_, err = clientGo.DynamicClient.Resource(mapping.Resource).Namespace(namespace).Create(context.TODO(), unstructuredObj, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("资源 %s 创建成功", unstructuredObj.GetName()), nil
}

func queryResource(namespace, resourceType string) (string, error) {
	clientGo, err := utils.NewClientGo(kubeconfig)
	if err != nil {
		return "", err
	}
	resourceType = strings.ToLower(resourceType)

	var gvr schema.GroupVersionResource
	switch resourceType {
	case "pod":
		gvr = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	case "service":
		gvr = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
	case "deployment":
		gvr = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	case "configmap":
		gvr = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	case "satefulset":
		gvr = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}
	case "secret":
		gvr = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}
	default:
		return "", fmt.Errorf("不支持的资源类型: %s", resourceType)
	}
	// 通过 dynamicClient 获取资源
	resourceList, err := clientGo.DynamicClient.Resource(gvr).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	result := ""
	for _, item := range resourceList.Items {
		result += fmt.Sprintf("资源名称: %s, 资源类型: %s\n", item.GetName(), resourceType)
	}
	return result, nil
}
func deleteResource(namespace, resourceType, resourceName string) (string, error) {
	clientGo, err := utils.NewClientGo(kubeconfig)
	if err != nil {
		return "", err
	}
	resourceType = strings.ToLower(resourceType)

	var gvr schema.GroupVersionResource
	switch resourceType {
	case "pod":
		gvr = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	case "service":
		gvr = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
	case "deployment":
		gvr = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	case "configmap":
		gvr = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	case "secret":
		gvr = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}
	case "satefulset":
		gvr = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}
	default:
		return "", fmt.Errorf("不支持的资源类型: %s (当前支持: pod/service/deployment/configmap/secret/satefulset)", resourceType)
	}
	// 处理默认命名空间
	if namespace == "" {
		namespace = "default"
	}

	// 执行删除操作
	err = clientGo.DynamicClient.Resource(gvr).Namespace(namespace).Delete(context.TODO(), resourceName, metav1.DeleteOptions{})
	if err != nil {
		// 通过错误信息内容判断
		if strings.Contains(err.Error(), "not found") {
			return "", fmt.Errorf("%s %s 在命名空间 %s 中不存在",
				resourceType, resourceName, namespace)
		}
		return "", fmt.Errorf("删除失败: %v", err)
	}
	return fmt.Sprintf("成功删除 %s/%s 于命名空间 %s", resourceType, resourceName, namespace), nil
}

func init() {
	askCmd.AddCommand(deepseekCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// deepseekCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// deepseekCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
