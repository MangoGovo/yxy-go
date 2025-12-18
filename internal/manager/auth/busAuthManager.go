package auth

import (
	"context"
	"errors"
	"net/url"
	"time"
	"yxy-go/internal/consts"
	"yxy-go/internal/svc"
	"yxy-go/internal/utils/yxyClient"
	"yxy-go/pkg/xerr"

	"github.com/go-resty/resty/v2"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

type BusAuthManager struct {
	logx.Logger
	ctx      context.Context
	svcCtx   *svc.ServiceContext
	cacheTTL time.Duration
}

func NewBusAuthManager(ctx context.Context, svcCtx *svc.ServiceContext) *BusAuthManager {
	return &BusAuthManager{
		ctx:      ctx,
		Logger:   logx.WithContext(ctx),
		svcCtx:   svcCtx,
		cacheTTL: 24 * time.Hour,
	}
}

type wxAuthResp struct {
	Token string `json:"token"`
}

// FetchAuthToken 发送请求获取AuthToken
func (l *BusAuthManager) FetchAuthToken(uid string) (string, error) {
	// 1. 鉴权请求
	resp, err := yxyClient.GetClient().R().
		SetQueryParams(map[string]string{
			"ymAppId":     consts.BUS_APPID,
			"callbackUrl": "https://api.pinbayun.com/api/v1/zjgd_interface/?schoolCode=10337",
			"authType":    "2",
			"authAppid":   consts.SCHOOL_CODE,
			"unionid":     uid,
			"schoolCode":  consts.SCHOOL_CODE,
		}).
		Get(consts.GET_BUS_AUTH_CODE_URL)
	if err != nil && !errors.Is(err, resty.ErrAutoRedirectDisabled) {
		return "", err
	}

	// 2. 获取code
	location := resp.RawResponse.Header.Get("Location")
	if location == "" {
		return "", xerr.WithCode(xerr.ErrUserNotFound, "用户不存在")
	}
	// 3. 获取corpcode
	resp, err = yxyClient.GetClient().R().Get(location)
	if err != nil && !errors.Is(err, resty.ErrAutoRedirectDisabled) {
		return "", err
	}
	location = resp.RawResponse.Header.Get("Location")
	u, err := url.Parse(location)
	if err != nil {
		return "", errors.New("鉴权获取corpcode失败:参数解析失败")
	}
	query := u.Query()
	corpcode := query.Get("corpcode")

	// 4. WX_Auth
	var fetchResp wxAuthResp
	_, headers := yxyClient.GetYxyBaseReqParam("")
	_, err = yxyClient.HttpSendPost(consts.GET_BUS_AUTH_TOKEN_URL, map[string]interface{}{
		"corpcode": corpcode,
		"openid":   2014120230,
	}, headers, &fetchResp)
	if err != nil {
		return "", err
	}
	return fetchResp.Token, nil
}

func (l *BusAuthManager) getCacheKey(uid string) string {
	return "bus:auth_token:" + uid
}

func (l *BusAuthManager) refreshCachedAuthToken(uid string) (string, error) {
	token, err := l.FetchAuthToken(uid)
	if err != nil {
		return "", err
	}
	key := l.getCacheKey(uid)
	l.svcCtx.Rdb.Set(l.ctx, key, token, cacheTTL)
	return token, nil
}

func (l *BusAuthManager) getCachedAuthToken(uid string) (string, error) {
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

func (l *BusAuthManager) WithAuthToken(uid string, fn func(token string) (any, error)) (any, error) {
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
