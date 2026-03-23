// utils/response.go, 统一 API 响应格式封装
package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 标准化的返回结构体
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}, msg string) {
	if msg == "" {
		msg = "success"
	}
	c.JSON(http.StatusOK, Response{
		Code: 200, // 业务状态码 200 代表成功
		Msg:  msg,
		Data: data,
	})
}

// Error 错误响应
func Error(c *gin.Context, httpStatus int, msg string) {
	c.JSON(httpStatus, Response{
		Code: httpStatus, // 业务状态码直接沿用 HTTP 状态码
		Msg:  msg,
		Data: nil,
	})
}
