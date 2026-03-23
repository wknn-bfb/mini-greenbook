// service/post_test.go
package service

import (
	"testing"

	"mini-greenbook/config"
	"mini-greenbook/model"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// TestMain 是测试的入口，所有测试函数执行前先跑这里
// 负责初始化数据库连接
func TestMain(m *testing.M) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("../") // service/ 的上一级就是项目根目录

	if err := viper.ReadInConfig(); err != nil {
		panic("测试环境读取配置文件失败: " + err.Error())
	}

	config.InitConfig()
	config.InitMySQL()
	config.InitRedis()

	// 自动迁移确保表结构存在
	config.DB.AutoMigrate(&model.User{}, &model.Post{})

	// 运行所有测试
	m.Run()
}

// TestCreatePost_Success 正常创建笔记
func TestCreatePost_Success(t *testing.T) {
	// 准备：清理可能存在的脏数据
	config.DB.Exec("DELETE FROM posts WHERE title = '测试标题'")

	// 执行
	err := CreatePost(1, "测试标题", "测试内容", "", "美食,探店")

	// 断言
	assert.NoError(t, err, "正常创建笔记不应该返回错误")

	// 验证数据库里确实有这条记录
	var post model.Post
	result := config.DB.Where("title = ?", "测试标题").First(&post)
	assert.NoError(t, result.Error, "数据库应该能查到这条笔记")
	assert.Equal(t, "测试内容", post.Content)
	assert.Equal(t, "美食,探店", post.Tags)

	// 清理测试数据
	config.DB.Unscoped().Delete(&post)
}

// TestToggleLike_FirstLike 第一次点赞
func TestToggleLike_FirstLike(t *testing.T) {
	// 准备：插入测试用的帖子和用户
	post := model.Post{Title: "点赞测试帖", Content: "内容", UserID: 1}
	config.DB.Create(&post)
	defer func() {
		// 先删中间表关联记录
		config.DB.Exec("DELETE FROM user_like_posts WHERE post_id = ?", post.ID)
		// 再删帖子
		config.DB.Unscoped().Delete(&post)
	}()

	// 执行第一次点赞
	isLiked, err := ToggleLike(1, post.ID)

	// 断言
	assert.NoError(t, err)
	assert.True(t, isLiked, "第一次点赞应该返回 true")

	// 验证中间表确实有记录
	var count int64
	config.DB.Table("user_like_posts").
		Where("post_id = ? AND user_id = ?", post.ID, 1).
		Count(&count)
	assert.Equal(t, int64(1), count, "中间表应该有一条点赞记录")
}

// TestToggleLike_CancelLike 再次点击取消点赞
func TestToggleLike_CancelLike(t *testing.T) {
	// 准备：插入帖子，并预先点一个赞
	post := model.Post{Title: "取消点赞测试帖", Content: "内容", UserID: 1}
	config.DB.Create(&post)
	defer func() {
		config.DB.Exec("DELETE FROM user_like_posts WHERE post_id = ?", post.ID)
		config.DB.Unscoped().Delete(&post)
	}()

	// 先点一次赞
	ToggleLike(1, post.ID)

	// 执行：再点一次，应该取消
	isLiked, err := ToggleLike(1, post.ID)

	// 断言
	assert.NoError(t, err)
	assert.False(t, isLiked, "第二次点击应该返回 false（取消点赞）")

	// 验证中间表记录已删除
	var count int64
	config.DB.Table("user_like_posts").
		Where("post_id = ? AND user_id = ?", post.ID, 1).
		Count(&count)
	assert.Equal(t, int64(0), count, "取消点赞后中间表应该没有记录")
}
