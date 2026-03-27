// cmd/main.go, 程序主入口
package main

import (
	"mini-greenbook/config"
	"mini-greenbook/model"
	"mini-greenbook/router"
	"net/http"
	_ "net/http/pprof" // 注册 pprof 的 HTTP 路由，下划线表示只要副作用
	"runtime"
	"time"

	"go.uber.org/zap"
)

func main() {
	// 1. 最先初始化基础配置和日志组件
	config.InitConfig()
	config.Log.Info("🚀 正在启动小绿书后端服务...")

	// 2. 初始化数据库连接
	config.InitMySQL()
	config.InitRedis()

	// 3. 自动迁移（同步表结构到 MySQL）
	config.Log.Info("🔄 正在同步数据库表结构...")
	err := config.DB.AutoMigrate(&model.User{}, &model.Post{})
	if err != nil {
		config.Log.Fatal("❌ 数据库表结构同步失败", zap.Error(err))
	}
	config.Log.Info("✅ 数据库表结构同步完成！")

	// 4.在独立端口暴露 pprof，不影响业务端口 8080
	go func() {
		config.Log.Info("pprof 监听 :6060")
		http.ListenAndServe(":6060", nil)
	}()

	// 5. 每30秒打印一次 goroutine 数量，监控有没有 goroutine 泄漏
	go func() {
		for {
			time.Sleep(30 * time.Second)
			config.Log.Info("goroutine 监控",
				zap.Int("count", runtime.NumGoroutine()))
		}
	}()

	// 6. 挂载路由并启动 Web 服务
	r := router.SetupRouter()
	config.Log.Info("🌐 服务启动成功，监听端口 8080...")
	if err := r.Run(":8080"); err != nil {
		config.Log.Fatal("❌ 服务器启动失败", zap.Error(err))
	}
}
