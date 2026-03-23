// service/llm_tag.go, 专职处理大模型 API 调用，生成标签
package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"mini-greenbook/config"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// GenerateTagsFromLLM 调用大模型 API 提取标签
func GenerateTagsFromLLM(content string) (string, error) {
	apiUrl := viper.GetString("llm.api_url")
	apiKey := viper.GetString("llm.api_key")
	modelName := viper.GetString("llm.model")

	// 1. 构造发给大模型的 Prompt（提示词）
	systemPrompt := "你是一个小红书爆款标签提取助手。请根据用户提供的笔记内容，提取3到5个核心分类标签。要求：只返回标签内容，用逗号分隔，绝对不要带有任何解释性的文字，不要带井号(#)。"

	// 构造 OpenAI 格式的请求体
	requestBody := map[string]interface{}{
		"model": modelName,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": content},
		},
		"temperature": 0.3, // 降低发散性，保证标签的稳定性
	}

	jsonBody, _ := json.Marshal(requestBody)

	// 2. 发起 HTTP 请求（三步走：构建请求对象-方法、url、body； 设置请求头； 发送请求）
	req, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonBody))
	if err != nil {
		config.Log.Error("构造 LLM 请求失败", zap.Error(err))
		return "", errors.New("AI 服务内部错误")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		config.Log.Error("请求 LLM 接口失败", zap.Error(err))
		return "", errors.New("网络请求失败，请稍后重试")
	}

	// 确保函数函数返回前一定关闭 HTTP请求流
	defer resp.Body.Close()

	// 3. 解析大模型返回的结果
	body, _ := io.ReadAll(resp.Body)

	// 如果非 200，说明 API Key 错或者余额不足
	if resp.StatusCode != http.StatusOK {
		config.Log.Error("LLM 返回非 200 错误", zap.String("response", string(body)))
		return "", errors.New("AI 生成失败，请检查配置")
	}

	// 临时结构体用于解析 OpenAI 格式的 JSON 响应
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		config.Log.Error("解析 LLM 响应失败", zap.Error(err))
		return "", errors.New("解析 AI 数据失败")
	}

	// 确保有返回结果
	if len(result.Choices) > 0 {
		return result.Choices[0].Message.Content, nil
	}

	return "", errors.New("AI 未返回有效内容")
}
