// controller/post.go, 负责处理笔记相关的 HTTP 请求
package controller

import (
	"net/http"
	"strconv"

	"mini-greenbook/service"
	"mini-greenbook/utils"

	"github.com/gin-gonic/gin"
)

// CreatePostReq 发布笔记的 JSON 参数结构
type CreatePostReq struct {
	Title    string `json:"title" binding:"required"`
	Content  string `json:"content" binding:"required"`
	ImageURL string `json:"image_url"` // 这里只接收前端传来的 OSS 链接
	Tags     string `json:"tags"`      // 用户最终确认的标签
}

// CreatePost 处理发布笔记请求
func CreatePost(c *gin.Context) {
	var req CreatePostReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "参数错误，标题和正文不能为空")
		return
	}

	// 【核心逻辑】：从上下文中提取鉴权中间件解析出来的 userID
	// c.MustGet 确保拿不到值时会直接引发 panic（但中间件已经拦截了非法请求，所以这里绝对安全）
	// .(uint) 是 Go 语言的类型断言，将万能接口类型 interface{} 转换为明确的 uint 类型
	userID := c.MustGet("userID").(uint)

	// 调用 Service 层入库
	if err := service.CreatePost(userID, req.Title, req.Content, req.ImageURL, req.Tags); err != nil {
		utils.Error(c, http.StatusInternalServerError, "发布笔记失败")
		return
	}

	utils.Success(c, nil, "发布成功！")
}

// GetPostList 处理获取笔记发现流请求
func GetPostList(c *gin.Context) {
	// 从URL接收参数，设置默认值
	cursorStr := c.DefaultQuery("cursor", "0")
	sizeStr := c.DefaultQuery("size", "10")

	// 将字符串转换为整数 (Atoi = ASCII to Integer)
	cursor, _ := strconv.Atoi(cursorStr)
	size, _ := strconv.Atoi(sizeStr)

	// 防止恶意传入负数导致 SQL 报错
	if cursor < 0 {
		cursor = 0
	}
	if size <= 0 || size > 100 {
		size = 10
	}

	// 调用 Service 层查询数据
	posts, nextCursor, err := service.GetPostList(uint(cursor), size)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "获取笔记列表失败")
		return
	}

	// 使用统一成功响应
	utils.Success(c, gin.H{
		"list":        posts,
		"next_cursor": nextCursor,
		"size":        size,
	}, "获取成功")
}

// ToggleLike 处理点赞/取消点赞请求
func ToggleLike(c *gin.Context) {
	// 从 URL 参数中获取笔记 ID: /v1/posts/:id/like
	postIDStr := c.Param("id")
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "无效的笔记 ID")
		return
	}

	// 从安检门中间件拿出现场用户的 ID
	userID := c.MustGet("userID").(uint)

	// 交给后厨（Service）去处理点赞逻辑
	isLiked, err := service.ToggleLike(userID, uint(postID))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "操作失败")
		return
	}

	// 智能返回文案
	msg := "点赞成功"
	if !isLiked {
		msg = "已取消点赞"
	}

	utils.Success(c, gin.H{"is_liked": isLiked}, msg)
}

// SearchPosts 处理笔记标签搜索请求
func SearchPosts(c *gin.Context) {
	tag := c.Query("tag")
	if tag == "" {
		utils.Error(c, http.StatusBadRequest, "搜索标签不能为空")
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	sizeStr := c.DefaultQuery("size", "10")
	page, _ := strconv.Atoi(pageStr)
	size, _ := strconv.Atoi(sizeStr)
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 10
	}

	posts, total, err := service.SearchPostsByTag(tag, page, size)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "搜索失败")
		return
	}

	utils.Success(c, gin.H{
		"list":  posts,
		"total": total,
		"page":  page,
		"size":  size,
	}, "搜索成功")
}

// GetHotList 处理获取热榜请求
func GetHotList(c *gin.Context) {
	sizeStr := c.DefaultQuery("size", "10")
	size, _ := strconv.Atoi(sizeStr)
	if size <= 0 || size > 100 {
		size = 10
	}

	posts, err := service.GetHotList(size)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "获取热榜失败")
		return
	}

	utils.Success(c, gin.H{"list": posts}, "获取成功")
}
