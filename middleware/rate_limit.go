// middleware/rate_limit.go, 基于 Redis ZSet 的滑动窗口限流中间件
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"mini-greenbook/config"
	"mini-greenbook/utils"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RateLimit 基于滑动窗口的限流中间件
// limit：时间窗口内最大请求次数
// window：时间窗口大小
func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background()

		userID := c.MustGet("userID").(uint)
		key := fmt.Sprintf("rate:sliding:ai:tags:%d", userID)
		now := time.Now().UnixMilli()
		windowStart := now - window.Milliseconds()

		pipe := config.RedisClient.Pipeline()

		// 1. 删除窗口之外的过期记录
		pipe.ZRemRangeByScore(ctx, key, "0",
			strconv.FormatInt(windowStart, 10))

		// 2. 统计当前窗口内的请求数
		pipe.ZCard(ctx, key)

		// 3. 把本次请求的时间戳写入 ZSet
		// member 用 time.Now().UnixNano() 保证唯一，避免同一毫秒内多次请求冲突
		pipe.ZAdd(ctx, key, redis.Z{
			Score:  float64(now),
			Member: time.Now().UnixNano(),
		})

		// 4. 设置 key 过期时间，窗口过后自动清理
		pipe.Expire(ctx, key, window)

		results, err := pipe.Exec(ctx)
		if err != nil {
			config.Log.Error("滑动窗口限流执行失败", zap.Error(err))
			// Redis 故障时放行，不能因限流组件故障影响正常业务
			c.Next()
			return
		}

		// 取第二步 ZCard 的结果：窗口内已有的请求数（不含本次）
		count := results[1].(*redis.IntCmd).Val()

		if count >= int64(limit) {
			config.Log.Info("触发限流",
				zap.Uint("userID", userID),
				zap.Int64("count", count),
			)
			utils.Error(c, http.StatusTooManyRequests,
				fmt.Sprintf("请求过于频繁，每分钟最多调用 %d 次，请稍后再试", limit))
			c.Abort()
			return
		}

		c.Next()
	}
}
