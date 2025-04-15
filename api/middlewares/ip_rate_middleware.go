package middlewares

import (
	"pledge-backend/api/common"
	"pledge-backend/api/common/statecode"
	"pledge-backend/api/models/response"
	"pledge-backend/db"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// 配置信息，偷个懒，直接写成常量了
	COUNT  = 5
	WINDOW = 60
	EXPIRE = 300
)

type IpRateMiddleware struct{}

// IP限流的中间件，这里利用redis的zset来实现（也可以使用令牌桶的方式）
//
//	1、每次访问时，判断范围内的访问次数是否超过上限，如果超过上限，限流
//	2、如果没有被限流，将时间窗口之前的数据移除，并且将当前
func (mw *IpRateMiddleware) Middleware(ctx *gin.Context) {
	// 获取IP
	ip := ctx.ClientIP()
	key := common.IP_RATE_LIMIT_KEY_PREFIX + ip
	// 获取当前时间戳
	now := time.Now()
	nowMill := now.UnixMilli()
	startMill := now.Add(-WINDOW * time.Second).UnixMilli()
	count, err := db.RedisZCount(key, startMill, nowMill)
	if err != nil {
		resp := response.Gin{Res: ctx}
		resp.Response(ctx, statecode.CommonErrServerErr, nil)
		ctx.Abort()
	}

	// 超限了
	if count >= COUNT {
		resp := response.Gin{Res: ctx}
		resp.Response(ctx, statecode.CommonErrServerErr, nil)
		ctx.Abort()
	}

	err = db.RedisZAdd(key, nowMill, nowMill, EXPIRE)
	if err != nil {
		resp := response.Gin{Res: ctx}
		resp.Response(ctx, statecode.CommonErrServerErr, nil)
		ctx.Abort()
	}
	// 查询区间内的访问次数
	count, err = db.RedisZCount(key, startMill, nowMill)
	if err != nil {
		resp := response.Gin{Res: ctx}
		resp.Response(ctx, statecode.CommonErrServerErr, nil)
		ctx.Abort()
	}

	// 删除窗口外的数据
	values, _ := db.RedisZRange(key, 0, startMill)
	if values != nil && len(values) > 0 {
		for _, value := range values {
			db.RedisZRem(key, value)
		}
	}

	ctx.Next()
}
