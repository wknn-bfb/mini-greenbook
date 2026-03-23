// service/post.go, 笔记相关的核心业务逻辑
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"strconv"
	"strings"

	"mini-greenbook/config"
	"mini-greenbook/model"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

// 定义包级变量 Singleflight 实例
var sfGroup singleflight.Group

// PostListResult 用于 Singleflight 包装返回结果的包级结构体
type PostListResult struct {
	Posts      []model.Post
	NextCursor uint // 游标分页不再需要 Total，改为返回下一页的游标
}

// HotListKey 热榜的 Redis ZSet Key
const HotListKey = "posts:hot"

// CreatePost 创建笔记入库
func CreatePost(userID uint, title, content, imageURL string, tags string) error {
	post := model.Post{
		UserID:   userID,
		Title:    title,
		Content:  content,
		ImageURL: imageURL,
		Tags:     tags,
	}

	// GORM 的 Create 方法会自动生成 INSERT SQL 语句
	if err := config.DB.Create(&post).Error; err != nil {
		config.Log.Error("写入笔记失败", zap.Error(err))
		return err
	}

	return nil
}

// GetPostList 游标分页获取笔记列表
//   - cursor=0（第一页）：直接查 MySQL，保证实时性，不走缓存
//   - cursor>0（翻页）：走 Redis 缓存，历史数据不变，缓存有效
//   - 防击穿：Singleflight 保证同一游标位置的并发请求只查一次 MySQL
//   - 防雪崩：缓存过期时间加随机抖动，避免同时失效
func GetPostList(cursor uint, size int) ([]model.Post, uint, error) {
	cacheKey := fmt.Sprintf("posts:cursor:%d:size:%d", cursor, size)
	ctx := context.Background()

	// 第一页：跳过缓存，直接查 MySQL
	if cursor == 0 {
		return queryFromDB(cursor, size, ctx, cacheKey, false) // false = 不写缓存
	}

	// 翻页：1. 先查 Redis 缓存
	cachedData, err := config.RedisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		// 2. 缓存命中：反序列化直接返回
		var cacheRes PostListResult
		if err := json.Unmarshal([]byte(cachedData), &cacheRes); err == nil {
			config.Log.Info("🚀 命中 Redis 缓存", zap.String("key", cacheKey))
			return cacheRes.Posts, cacheRes.NextCursor, nil
		}
	}

	// 3. 缓存未命中：Singleflight 保护下查 MySQL
	return queryFromDB(cursor, size, ctx, cacheKey, true) // true = 查完写缓存
}

// queryFromDB 真正执行 MySQL 查询的内部函数
func queryFromDB(cursor uint, size int, ctx context.Context, cacheKey string, writeCache bool) ([]model.Post, uint, error) {
	// writeCache = true 时查完写入 Redis（翻页场景）
	// writeCache = false 时只查不缓存（第一页场景）
	// Singleflight：同一个 cacheKey 的并发请求只有第一个真正执行 其余请求阻塞等待，最终拿到同一份结果
	v, err, shared := sfGroup.Do(cacheKey, func() (interface{}, error) {
		config.Log.Info("🐢 查询 MySQL", zap.String("key", cacheKey))

		var dbPosts []model.Post
		query := config.DB.Model(&model.Post{})

		// cursor=0 时不加条件，从最大 id（最新帖子）开始取，否则 WHERE id < cursor
		if cursor > 0 {
			query = query.Where("id < ?", cursor)
		}

		if err := query.Order("id desc").Limit(size).Find(&dbPosts).Error; err != nil {
			return nil, err
		}

		// 计算下一页游标：本页最后一条记录的 ID
		// len=0 说明已经没有数据，NextCursor=0 通知前端停止加载
		var dbNextCursor uint = 0
		if len(dbPosts) > 0 {
			dbNextCursor = dbPosts[len(dbPosts)-1].ID
		}

		res := PostListResult{Posts: dbPosts, NextCursor: dbNextCursor}

		// 只有翻页场景才写缓存 把完整的 PostListResult 序列化存入 Redis
		if writeCache {
			if jsonData, err := json.Marshal(res); err == nil {
				// 随机抖动：60~90 秒之间随机，防止不同页的缓存同时过期（防雪崩）
				expiration := time.Duration(60+rand.Intn(30)) * time.Second
				config.RedisClient.Set(ctx, cacheKey, jsonData, expiration)
			}
		}

		return res, nil
	})

	if err != nil {
		return nil, 0, err
	}

	// Singleflight 返回 interface{}，断言回具名类型
	res := v.(PostListResult)

	if shared {
		config.Log.Info("🛡️ Singleflight 拦截并发请求", zap.String("key", cacheKey))
	}

	return res.Posts, res.NextCursor, nil
}

// ToggleLike 切换点赞状态 (点赞/取消点赞)
// 查和写之间有时间窗口，并发时两个请求都查到 count=0，都去插入。
// "查-写"之间存在竞态，引入分布式锁把这两步变成原子操作
func ToggleLike(userID uint, postID uint) (bool, error) {
	ctx := context.Background()

	// 1. 构造锁的 key，粒度精确到"同一用户+同一笔记"
	// 因为不同用户点不同笔记互不影响，只有同一用户对同一笔记的并发请求才会竞争锁
	lockKey := fmt.Sprintf("lock:like:%d:%d", userID, postID)

	// 2. 尝试获取锁
	// SetNX 原子操作，同时只有一个请求能设置成功，5秒过期，防止永不释放
	lockValue := uuid.New().String() // 用 UUID 作为锁的值
	ok, err := config.RedisClient.SetNX(ctx, lockKey, lockValue, 5*time.Second).Result()
	if err != nil {
		config.Log.Error("获取分布式锁失败", zap.Error(err))
		return false, errors.New("系统繁忙，请稍后重试")
	}
	if !ok {
		// 抢锁失败，说明同一用户正在操作同一篇笔记，直接拒绝
		return false, errors.New("操作太频繁，请稍后再试")
	}

	// 3. 确保函数返回前释放锁
	// Lua 脚本：判断是不是自己的锁，是才删，原子操作
	defer func() {
		luaScript := redis.NewScript(`
            if redis.call("GET", KEYS[1]) == ARGV[1] then
                return redis.call("DEL", KEYS[1])
            else
                return 0
            end
        `)
		luaScript.Run(ctx, config.RedisClient, []string{lockKey}, lockValue)
	}()

	// 4. 持锁后执行查询和写入，此时不会有并发竞争
	post := model.Post{}
	post.ID = postID
	user := model.User{}
	user.ID = userID

	var count int64
	// 去底层中间表查一下，这俩人之前有没有过交集
	config.DB.Table("user_like_posts").Where("post_id = ? AND user_id = ?", postID, userID).Count(&count)

	if count > 0 {
		// 已经点过赞 -> 执行【取消点赞】(从关联表删除对应记录)
		err := config.DB.Model(&post).Association("LikedByUsers").Delete(&user)
		go updateHotScore(postID) // 异步更新热度，不阻塞点赞响应
		return false, err         // 返回 false 代表当前变成了未点赞状态
	} else {
		// 没点过赞 -> 执行【点赞】(向关联表插入一条记录)
		err := config.DB.Model(&post).Association("LikedByUsers").Append(&user)
		go updateHotScore(postID) // 异步更新热度，不阻塞点赞响应
		return true, err          // 返回 true 代表点赞成功
	}
}

// SearchPostsByTag 根据标签模糊搜索笔记列表
func SearchPostsByTag(tag string, page, size int) ([]model.Post, int64, error) {
	var posts []model.Post
	var total int64

	offset := (page - 1) * size

	// 模糊查询：tags LIKE '%美食%'
	query := config.DB.Model(&model.Post{}).Where("tags LIKE ?", "%"+tag+"%")

	query.Count(&total)
	err := query.Order("created_at desc").Offset(offset).Limit(size).Find(&posts).Error

	// 注意：搜索接口通常条件组合极其复杂，一般不强求放入 Redis，直接查库即可（或使用 ElasticSearch）
	return posts, total, err
}

// updateHotScore 更新笔记在热榜 ZSet 中的分数
// 每次点赞/取消点赞后调用
func updateHotScore(postID uint) {
	ctx := context.Background()

	// 查当前点赞数
	var likeCount int64
	config.DB.Table("user_like_posts").
		Where("post_id = ?", postID).
		Count(&likeCount)

	// 查发帖时间
	var post model.Post
	if err := config.DB.Select("created_at").First(&post, postID).Error; err != nil {
		config.Log.Error("查询帖子时间失败", zap.Error(err))
		return
	}

	// 计算热度值：点赞数×10 + 时间衰减因子
	hoursSincePost := time.Since(post.CreatedAt).Hours()
	score := float64(likeCount)*10 + 1000/(hoursSincePost+2)

	// 写入 ZSet
	config.RedisClient.ZAdd(ctx, HotListKey, redis.Z{
		Score:  score,
		Member: postID,
	})

	config.Log.Info("更新热度分数",
		zap.Uint("postID", postID),
		zap.Float64("score", score),
	)
}

// GetHotList 从 ZSet 获取热榜笔记
func GetHotList(size int) ([]model.Post, error) {
	ctx := context.Background()

	// 从 ZSet 按分数从高到低取 size 条，拿到的是 postID 列表
	results, err := config.RedisClient.ZRevRange(ctx, HotListKey, 0, int64(size-1)).Result()
	if err != nil || len(results) == 0 {
		// ZSet 为空时降级查 MySQL，按发布时间排序
		config.Log.Info("热榜 ZSet 为空，降级查 MySQL")
		var posts []model.Post
		err = config.DB.Order("created_at desc").Limit(size).Find(&posts).Error
		return posts, err
	}

	// 把 string 类型的 postID 列表转成 uint
	postIDs := make([]uint, 0, len(results))
	for _, r := range results {
		id, _ := strconv.ParseUint(r, 10, 64)
		postIDs = append(postIDs, uint(id))
	}

	// 按 postID 列表查 MySQL 拿完整数据
	var posts []model.Post
	// IN 查询不保证顺序，用 ORDER BY FIELD 保持 ZSet 的热度顺序
	config.DB.Where("id IN ?", postIDs).
		Order(fmt.Sprintf("FIELD(id, %s)",
			strings.Join(results, ","))).
		Find(&posts)

	return posts, nil
}
