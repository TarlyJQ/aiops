/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/TarlyJQ/aiops/k8scopilot/cmd/utils"
	"github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// eventCmd represents the event command

// 修改后的 eventCmd
var eventCmd = &cobra.Command{
	Use:   "event",
	Short: "交互式分析集群异常事件",
	Run: func(cmd *cobra.Command, args []string) {
		// 获取问题 Pod 列表
		pods, err := getProblemPods()
		if err != nil {
			fmt.Println("获取集群状态失败:", err)
			return
		}

		if len(pods) == 0 {
			fmt.Println("✅ 当前集群运行正常")
			return
		}

		// 交互选择
		selectedPod, err := selectPod(pods)
		if err != nil {
			fmt.Println("选择无效")
			return
		}

		// 执行分析
		result, err := analyzeSinglePod(selectedPod)
		if err != nil {
			fmt.Println("分析失败:", err)
			return
		}

		fmt.Println("\n分析结果：")
		fmt.Println(result)
	},
}

// 新增结构体存储 Pod 信息
type PodIssue struct {
	Name      string
	Namespace string
	Events    []string
	Logs      string
}

// 步骤1：获取问题 Pod 列表
func getProblemPods() ([]PodIssue, error) {
	clientGo, err := utils.NewClientGo(kubeconfig)
	if err != nil {
		return nil, err
	}

	var podIssues []PodIssue

	events, err := clientGo.ClientSet.CoreV1().Events("").List(context.TODO(), metav1.ListOptions{
		FieldSelector: "type=Warning",
	})
	if err != nil {
		return nil, err
	}

	for _, event := range events.Items {
		if event.InvolvedObject.Kind != "Pod" {
			continue
		}

		podName := event.InvolvedObject.Name
		namespace := event.InvolvedObject.Namespace

		// 去重处理
		exists := false
		for _, p := range podIssues {
			if p.Name == podName && p.Namespace == namespace {
				exists = true
				break
			}
		}
		if exists {
			continue
		}

		// 获取基础信息
		podIssue := PodIssue{
			Name:      podName,
			Namespace: namespace,
			Events:    []string{event.Message},
		}

		// 获取日志（保留核心错误信息）
		if pod, err := clientGo.ClientSet.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{}); err == nil {
			if pod.Status.Phase == corev1.PodRunning {
				if logs, err := getLast100LinesLog(namespace, podName); err == nil {
					podIssue.Logs = logs
				}
			}
		}

		podIssues = append(podIssues, podIssue)
	}

	return podIssues, nil
}

func ptr[T any](v T) *T {
	return &v
}

// 获取日志最后100行（控制长度）
func getLast100LinesLog(namespace, podName string) (string, error) {
	clientGo, err := utils.NewClientGo(kubeconfig)
	if err != nil {
		return "", err
	}

	logOptions := &corev1.PodLogOptions{
		TailLines: ptr(int64(100)), // 仅获取最后100行
	}

	req := clientGo.ClientSet.CoreV1().Pods(namespace).GetLogs(podName, logOptions)
	podLogs, err := req.Stream(context.TODO())
	if err != nil {
		return "", err
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(podLogs); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// 步骤2：交互式选择
func selectPod(pods []PodIssue) (PodIssue, error) {
	fmt.Println("发现异常 Pod 列表：")
	for i, pod := range pods {
		fmt.Printf("[%d] %s/%s - 事件: %s\n",
			i+1,
			pod.Namespace,
			pod.Name,
			strings.Join(pod.Events, ", "),
		)
	}

	fmt.Print("\n请选择要分析的 Pod 序号 (0 退出): ")

	var choice int
	_, err := fmt.Scanf("%d", &choice)
	if err != nil || choice < 1 || choice > len(pods) {
		return PodIssue{}, errors.New("无效输入")
	}

	return pods[choice-1], nil
}

// 步骤3：发送单个 Pod 分析请求
func analyzeSinglePod(pod PodIssue) (string, error) {
	client, err := utils.NewOpenAIClient()
	if err != nil {
		return "", err
	}

	// 构造精炼提示词
	prompt := fmt.Sprintf(`请分析以下 Kubernetes Pod 问题：
Pod: %s/%s

事件列表:
%s

相关日志（最后100行）:
%s

请按以下格式响应：
1. 问题诊断（简明扼要）
2. 解决步骤（带具体命令）
3. 相关参考链接`,
		pod.Namespace, pod.Name,
		strings.Join(pod.Events, "\n- "),
		pod.Logs,
	)

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: "你是一个 Kubernetes 专家，请用简洁的技术语言分析问题",
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		},
	}

	resp, err := client.Client.CreateChatCompletion(
		context.TODO(),
		openai.ChatCompletionRequest{
			Model:     openai.GPT4o,
			Messages:  messages,
			MaxTokens: 500, // 限制响应长度
		},
	)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", errors.New("未收到有效响应")
	}

	return resp.Choices[0].Message.Content, nil
}

func getPodEventAndLogs() (map[string][]string, error) {
	clientGo, err := utils.NewClientGo(kubeconfig)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]string)

	// 1. 查询所有命名空间的 Warning 事件
	events, err := clientGo.ClientSet.CoreV1().Events("").List(context.TODO(), metav1.ListOptions{
		FieldSelector: "type=Warning",
	})
	if err != nil {
		return nil, err
	}

	for _, event := range events.Items {
		if event.InvolvedObject.Kind != "Pod" {
			continue
		}

		podName := event.InvolvedObject.Name
		namespace := event.InvolvedObject.Namespace
		message := event.Message

		// 2. 初始化日志条目
		entries := []string{
			fmt.Sprintf("Event: %s", message),
			fmt.Sprintf("Namespace: %s", namespace),
		}

		// 3. 获取 Pod 状态
		pod, err := clientGo.ClientSet.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			entries = append(entries, fmt.Sprintf("获取Pod状态失败: %v", err))
			result[podName] = entries
			continue
		}

		// 4. 只有 Running 状态才获取日志
		if pod.Status.Phase == corev1.PodRunning {
			logOption := &corev1.PodLogOptions{}
			req := clientGo.ClientSet.CoreV1().Pods(namespace).GetLogs(podName, logOption)
			podLogs, err := req.Stream(context.TODO())
			if err != nil {
				entries = append(entries, fmt.Sprintf("日志获取失败: %v", err))
			} else {
				defer podLogs.Close()
				buf := new(bytes.Buffer)
				if _, err := buf.ReadFrom(podLogs); err != nil {
					entries = append(entries, fmt.Sprintf("日志读取失败: %v", err))
				} else {
					entries = append(entries, fmt.Sprintf("日志内容:\n%s", buf.String()))
				}
			}
		} else {
			entries = append(entries, fmt.Sprintf("Pod状态: %s (无法获取日志)", pod.Status.Phase))
		}

		result[podName] = entries
	}

	return result, nil
}

func init() {
	analyzeCmd.AddCommand(eventCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// eventCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// eventCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
