package auth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"yxy-go/internal/consts"
	"yxy-go/internal/svc"
	"yxy-go/internal/utils/yxyClient"
	"yxy-go/pkg/xerr"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

type getAuthTokenResp struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Data       any    `json:"data"`
	Success    bool   `json:"success"`
}

type ElectricityAuthManager struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewElectricityAuthManager(ctx context.Context, svcCtx *svc.ServiceContext) *ElectricityAuthManager {
	return &ElectricityAuthManager{
		ctx:    ctx,
		Logger: logx.WithContext(ctx),
		svcCtx: svcCtx,
	}
}

// FetchAuthToken 发送请求获取AuthToken
func (l *ElectricityAuthManager) FetchAuthToken(uid string) (string, error) {
	_, yxyHeaders := yxyClient.GetYxyBaseReqParam("")
	yxyReq := map[string]string{
		"bindSkip":    "1",
		"authType":    "2",
		"ymAppId":     consts.ELECTRICTY_APPID,
		"callbackUrl": consts.APPLICATION_URL + "/",
		"unionid":     uid,
		"schoolCode":  consts.SCHOOL_CODE,
		"ymAuthToken": "",
	}

	client := yxyClient.GetClient()
	r, err := client.R().
		SetHeaders(yxyHeaders).
		SetQueryParams(yxyReq).
		Get(consts.GET_AUTH_CODE_URL)
	if r == nil || (err != nil && r.StatusCode() != 302) {
		l.Errorf("yxyClient.HttpSendPost err: %v , [%s]", err, consts.GET_AUTH_TOKEN)
		return "", xerr.WithCode(xerr.ErrHttpClient, err.Error())
	}

	location := r.Header().Get("Location")
	if location == "" {
		if strings.Contains(r.String(), "用户不存在") {
			return "", xerr.WithCode(xerr.ErrUserNotFound, fmt.Sprintf("User not found, UID: %v", uid))
		}
		return "", xerr.WithCode(xerr.ErrUnknown, fmt.Sprintf("yxy response: %v", r))
	}
	// hack 掉路由 hash模式 下url中的 /#/ 便于 query 参数提取
	location = strings.ReplaceAll(location, "#/", "")
	parsedURL, _ := url.Parse(location)
	ymCode := parsedURL.Query().Get("ymCode")

	var authResp getAuthTokenResp

	r, err = yxyClient.HttpSendPost(consts.GET_AUTH_TOKEN_URL,
		map[string]interface{}{
			"authType": "2",
			"code":     ymCode,
		}, yxyHeaders, &authResp)
	if err != nil {
		return "", err
	}

	if authResp.StatusCode != 0 {
		return "", xerr.WithCode(xerr.ErrUnknown, fmt.Sprintf("yxy response: %v", r))
	}
	var shiroJID string
	for _, cookie := range r.Cookies() {
		if cookie.Name == "shiroJID" {
			shiroJID = cookie.Value
			// 这里不break是因为会有多个重复的 shiroJID 要拿到最后一个
			// break
		}
	}
	l.Logger.Infof("%s获取token成功", uid)
	return shiroJID, nil
}

// getCacheKey 获取缓存token的key
func (l *ElectricityAuthManager) getCacheKey(uid string) string {
	// TODO(typo) 考虑后续修改为 auth_token:electricity:uid
	return "elec:auth_token:" + uid
}

// refreshCachedAuthToken 刷新缓存中的AuthToken
func (l *ElectricityAuthManager) refreshCachedAuthToken(uid string) (string, error) {
	token, err := l.FetchAuthToken(uid)
	if err != nil {
		return "", err
	}
	key := l.getCacheKey(uid)
	l.svcCtx.Rdb.Set(l.ctx, key, token, cacheTTL)
	return token, nil
}

// getCachedAuthToken 获取authToken, 优先从缓存中获取
func (l *ElectricityAuthManager) getCachedAuthToken(uid string) (string, error) {
	key := l.getCacheKey(uid)
	token, err := l.svcCtx.Rdb.Get(l.ctx, key).Result()
	if err == nil {
		return token, nil
	}

	if errors.Is(err, redis.Nil) {
		return l.refreshCachedAuthToken(uid)
	} else {
		return "", errors.New("获取缓存Token失败, redis异常")
	}
}

// WithAuthToken 包装需要使用token的业务函数, 只需要将其作为回调传入, 以下处理函数会自动处理token的获取和缓存, 并将token注入业务函数
func (l *ElectricityAuthManager) WithAuthToken(uid string, fn func(token string) (any, error)) (any, error) {
	// 1. 从缓存获取 token
	token, err := l.getCachedAuthToken(uid)
	if err != nil {
		return nil, err
	}

	// 2. 调用回调函数
	result, err := fn(token)
	if err == nil {
		l.Logger.Debugf("%s成功命中token缓存", uid)
		return result, nil
	}

	// 3. token 失效
	l.Logger.Errorf("token: %s 失效, 刷新token", token)
	if token, err = l.refreshCachedAuthToken(uid); err != nil {
		return nil, err
	}

	return fn(token)
}
