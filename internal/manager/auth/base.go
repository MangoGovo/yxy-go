package auth

import "time"

type AuthManager interface {
	// FetchAuthToken 发送请求获取AuthToken
	FetchAuthToken(uid string) (string, error)

	// getCacheKey 获取缓存token的key
	getCacheKey(uid string) string

	// refreshCachedAuthToken 刷新缓存中的AuthToken
	refreshCachedAuthToken(uid string) (string, error)

	// getCachedAuthToken 获取authToken, 优先从缓存中获取
	getCachedAuthToken(uid string) (string, error)

	// WithAuthToken 包装需要使用token的业务函数, 只需要将其作为回调传入, 以下处理函数会自动处理token的获取和缓存, 并将token注入业务函数
	WithAuthToken(uid string, fn func(token string) (any, error)) (any, error)
}

const cacheTTL = 24 * time.Hour

// 编译期断言, 检查接口是否全部实现
var _ AuthManager = (*BusAuthManager)(nil)
var _ AuthManager = (*ElectricityAuthManager)(nil)
