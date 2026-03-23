// router/router.go, 统一的 API 路由管理中心
package router

import (
	"net/http"
	"time"

	"mini-greenbook/controller"
	"mini-greenbook/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// 测试接口
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong! 数据库与 Web 服务已就绪"})
	})

	// 建立 API 版本分组
	v1 := r.Group("/v1")
	{
		// 1. 公共路由分组（不需要登录就能访问）
		auth := v1.Group("/auth")
		{
			auth.POST("/register", controller.Register) // 注册
			auth.POST("/login", controller.Login)       // 登录
		}

		// 发现流列表 搜索 热榜 是公开的
		v1.GET("/posts", controller.GetPostList)
		v1.GET("/posts/search", controller.SearchPosts)
		v1.GET("/posts/hot", controller.GetHotList)

		// 2. 受保护路由分组（必须携带合法的 JWT Token 才能访问）
		// 使用 Use() 挂载我们自己写的安检中间件
		protected := v1.Group("/")
		protected.Use(middleware.JWTAuth())
		{
			protected.POST("/posts", controller.CreatePost) // 发布笔记

			// 点赞/取消点赞笔记 (注意这里的 :id 是动态路由参数) 这里的id是post_id
			protected.POST("/posts/:id/like", controller.ToggleLike)

			// 调用 AI 生成标签，必须消耗自己的账号登录凭证。滑动窗口限流
			protected.POST("/ai/tags",
				middleware.RateLimit(3, time.Minute), // 每分钟最多5次
				controller.GenerateTags,
			)
		}
	}

	return r
}
