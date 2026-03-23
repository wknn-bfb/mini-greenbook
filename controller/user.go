// controller/user.go, 负责处理用户相关的 HTTP 请求
package controller

import (
	"net/http"

	"mini-greenbook/service"
	"mini-greenbook/utils"

	"github.com/gin-gonic/gin"
)

// RegisterReq 定义前端传过来的 JSON 数据结构
// binding 标签是 Gin 的参数校验功能
type RegisterReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"` // 密码至少 6 位
}

// LoginReq 登录请求参数
type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Register 处理用户注册请求
func Register(c *gin.Context) {
	var req RegisterReq

	// 1. 绑定并校验 JSON 参数
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "参数错误：用户名不能为空，且密码至少6位")
		return
	}

	// 2. 调用 Service 层执行真正的注册逻辑
	if err := service.RegisterUser(req.Username, req.Password); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// 3. 注册成功，返回统一格式的 JSON 响应
	utils.Success(c, nil, "注册成功，欢迎加入小绿书！")
}

// Login 处理用户登录请求
func Login(c *gin.Context) {
	var req LoginReq

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "参数错误：用户名或密码为空")
		return
	}

	// 调用 Service 执行登录并获取 Token
	token, err := service.LoginUser(req.Username, req.Password)
	if err != nil {
		// 登录失败（如密码错误、用户不存在）
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	// 登录成功，将 Token 返回给前端
	utils.Success(c, gin.H{"token": token}, "登录成功")
}
