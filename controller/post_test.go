package controller

import (
	"mini-greenbook/config"
	"mini-greenbook/model"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// 1. 初始化配置路径（../ 代表返回根目录找 config.yaml）
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("../")

	if err := viper.ReadInConfig(); err != nil {
		panic("测试环境读取配置文件失败: " + err.Error())
	}

	// 2. 初始化核心组件（确保 config.Log 不再是 nil）
	config.InitConfig()
	config.InitMySQL()
	config.InitRedis()

	// 3. 同步表结构（可选）
	config.DB.AutoMigrate(&model.User{}, &model.Post{})

	// 4. 运行所有测试并退出
	os.Exit(m.Run())
}

func TestGetPostList_Controller(t *testing.T) {
	// 1. 初始化 Gin
	r := gin.Default()

	//调用的是获取列表的函数
	r.GET("/v1/posts", GetPostList)

	// 2. 构造请求
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/posts?size=10", nil)

	// 3. 执行
	r.ServeHTTP(w, req)

	// 4. 断言
	assert.Equal(t, http.StatusOK, w.Code) // 应该拿到 200
	assert.Contains(t, w.Body.String(), "获取成功")
}
