// middleware/auth.go, JWT 鉴权拦截器
package middleware

import (
	"net/http"
	"strings"

	"mini-greenbook/utils"

	"github.com/gin-gonic/gin"
)

// JWTAuth 这是一个 Gin 中间件函数
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从 HTTP 请求头中获取 Authorization 字段
		authHeader := c.Request.Header.Get("Authorization")

		// 如果没有传 Token，或者格式不符合 "Bearer xxxx" 的标准
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			utils.Error(c, http.StatusUnauthorized, "请求未携带合法的 Token，无权访问")
			// Abort() 会立刻终止请求，不再交给后面的代码处理
			c.Abort()
			return
		}

		// 2. 提取出真正的 Token 字符串（去掉 "Bearer " 前缀）
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// 3. 解析并校验 Token
		claims, err := utils.ParseToken(tokenString)
		if err != nil {
			utils.Error(c, http.StatusUnauthorized, "Token 无效或已过期，请重新登录")
			c.Abort()
			return
		}

		// 4. 【核心】校验通过 将解析出的 UserID 存入 Gin 的上下文中
		// 这样后续的接口（如发布笔记）就可以直接从上下文里拿到“是谁发起的请求”了
		c.Set("userID", claims.UserID)

		// 5. 放行，继续执行后续的处理函数
		c.Next()
	}
}
