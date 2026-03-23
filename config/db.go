// config/db.go, 数据库与 Redis 连接配置
package config

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	DB          *gorm.DB
	RedisClient *redis.Client
	Log         *zap.Logger // 全局日志实例
)

// InitConfig 初始化配置与日志引擎
func InitConfig() {
	// 1. 初始化 Zap 日志 (开发模式，带颜色和调用栈)
	Log, _ = zap.NewDevelopment()

	// 2. 初始化 Viper 读取配置文件
	viper.SetConfigName("config") // 文件名为 config
	viper.SetConfigType("yaml")   // 格式为 yaml
	viper.AddConfigPath(".")      // 在项目根目录查找

	if err := viper.ReadInConfig(); err != nil {
		Log.Fatal("❌ 读取配置文件失败", zap.Error(err))
	}
}

// InitMySQL 初始化 MySQL 连接
func InitMySQL() {
	// 从 Viper 动态读取 DSN
	dsn := viper.GetString("mysql.dsn")

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		Log.Fatal("❌ MySQL 连接失败", zap.Error(err))
	}

	DB = db
	Log.Info("✅ MySQL 连接成功！")
}

// InitRedis 初始化 Redis 连接
func InitRedis() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     viper.GetString("redis.addr"),
		Password: viper.GetString("redis.password"),
		DB:       viper.GetInt("redis.db"),
	})

	// 测试 PING
	_, err := RedisClient.Ping(context.Background()).Result()
	if err != nil {
		Log.Fatal("❌ Redis 连接失败", zap.Error(err))
	}

	Log.Info("✅ Redis 连接成功！")
}
