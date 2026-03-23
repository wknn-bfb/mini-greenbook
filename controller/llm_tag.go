// controller/llm_tag.go, 负责处理 AI 生成标签相关的 HTTP 请求
package controller

import (
	"net/http"

	"mini-greenbook/service"
	"mini-greenbook/utils"

	"github.com/gin-gonic/gin"
)

type GenerateTagsReq struct {
	Content string `json:"content" binding:"required"`
}

// GenerateTags 调用大模型生成标签
func GenerateTags(c *gin.Context) {
	var req GenerateTagsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "参数错误，正文内容不能为空")
		return
	}

	// 拦截过短的内容（没必要浪费 AI 的 Token 去分析几个字）
	if len(req.Content) < 10 {
		utils.Error(c, http.StatusBadRequest, "笔记内容过短，AI 无法有效提取标签，请多写一点吧")
		return
	}

	// 调用 Service
	tags, err := service.GenerateTagsFromLLM(req.Content)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, gin.H{"tags": tags}, "AI 标签生成成功")
}
