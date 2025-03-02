/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/TarlyJQ/aiops/k8scopilot/cmd/utils"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/spf13/cobra"
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

func sanitizeYAML(raw string) string {
	// 强化版正则表达式（处理所有边界情况）
	re := regexp.MustCompile(
		`(?m)(^|\n)` + // 匹配开头或换行
			`\x60{3}(yaml)?` + // 匹配 ``` 或 ```yaml
			`(\n|$)` + // 匹配后续换行或结尾
			`|(\n?)\x60{3}$`, // 专门处理结尾的 ```
	)

	// 两阶段清理
	step1 := re.ReplaceAllString(raw, "")

	// 额外处理可能的残留
	return strings.TrimSpace(
		strings.TrimPrefix(
			strings.TrimSuffix(step1, "`"),
			"`",
		),
	)
}

func functionCalling(input string, client *utils.OpenAI) string {
	// 定义第一个函数 ，生成 K8s YAML 并且部署资源用的
	f1 := openai.FunctionDefinition{
		Name:        "generateAndDeployK8sYaml",
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
		Name:        "queryK8sResource",
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
		Name:        "generateAndDeployK8sYaml",
		Description: "生成 Kubernetes YAML 文件",
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
			Model:    "gpt-3.5-turbo",
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
	// dialogue = append(dialogue, msg)
	return fmt.Sprintf("OpenAI 希望能请求函数 %s，参数 %s", msg.ToolCalls[0].Function.Name, msg.ToolCalls[0].Function.Arguments)
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
