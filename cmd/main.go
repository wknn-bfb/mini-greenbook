// cmd/main.go, 程序主入口
package main

import (
	"mini-greenbook/config"
	"mini-greenbook/model"
	"mini-greenbook/router"

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

	// 4. 挂载路由并启动 Web 服务
	r := router.SetupRouter()

	config.Log.Info("🌐 服务启动成功，监听端口 8080...")
	if err := r.Run(":8080"); err != nil {
		config.Log.Fatal("❌ 服务器启动失败", zap.Error(err))
	}
}
