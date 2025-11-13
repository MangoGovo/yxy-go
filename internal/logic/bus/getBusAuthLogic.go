package bus

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"yxy-go/internal/consts"
	"yxy-go/internal/svc"
	"yxy-go/internal/types"
	"yxy-go/internal/utils/yxyClient"
	"yxy-go/pkg/xerr"

	"github.com/PuerkitoBio/goquery"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetBusAuthLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetBusAuthLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetBusAuthLogic {
	return &GetBusAuthLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

type GetBusAuthTokenYxyResp struct {
	Token string `json:"token"`
}

func (l *GetBusAuthLogic) GetBusAuth(req *types.GetBusAuthReq) (resp *types.GetBusAuthResp, err error) {
	_, yxyHeaders := yxyClient.GetYxyBaseReqParam("")
	yxyReq := map[string]string{
		"authType":    "2",
		"ymAppId":     "2011112043190345310",
		"authAppid":   "10337",
		"callbackUrl": consts.BUS_URL + "/api/v1/zjgd_interface/?schoolCode=10337",
		"unionid":     req.UID,
		"schoolCode":  consts.SCHOOL_CODE,
	}

	client := yxyClient.GetClient()
	r, err := client.R().
		SetHeaders(yxyHeaders).
		SetQueryParams(yxyReq).
		Get(consts.GET_BUS_AUTH_CODE_URL)
	if err != nil && r.StatusCode() != 302 && r.StatusCode() != 200 {
		return nil, xerr.WithCode(xerr.ErrHttpClient, err.Error())
	}

	location := r.Header().Get("Location")
	if location == "" {
		if strings.Contains(r.String(), "用户不存在") {
			return nil, xerr.WithCode(xerr.ErrUserNotFound, fmt.Sprintf("User not found, UID: %v", req.UID))
		}

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(r.String()))
		if err != nil {
			return nil, xerr.WithCode(xerr.ErrUnknown, err.Error())
		}
		stateCode := doc.Find("input.stateCode").AttrOr("value", "")
		if stateCode == "" {
			return nil, xerr.WithCode(xerr.ErrUnknown, fmt.Sprintf("stateCode is empty, UID: %v", req.UID))
		}

		r, err = client.R().
			SetHeaders(yxyHeaders).
			SetQueryParams(map[string]string{
				"stateCode": stateCode,
				"appid":     "2011112043190345310",
			}).Get(consts.GET_BUS_ACCESS_URL)
		if err != nil && r.StatusCode() != 302 {
			return nil, xerr.WithCode(xerr.ErrHttpClient, err.Error())
		}

		location = r.Header().Get("Location")
		if location == "" {
			return nil, xerr.WithCode(xerr.ErrUnknown, fmt.Sprintf("yxy response: %v", r))
		}
	}

	r, err = client.R().
		SetHeaders(yxyHeaders).
		Get(location)
	if err != nil && r.StatusCode() != 302 {
		return nil, xerr.WithCode(xerr.ErrHttpClient, err.Error())
	}

	location = r.Header().Get("Location")
	if location == "" {
		return nil, xerr.WithCode(xerr.ErrUnknown, fmt.Sprintf("yxy response: %v", r))
	}
	parsedURL, _ := url.Parse(location)
	queryParams := parsedURL.Query()
	openid := queryParams.Get("openid")
	corpcode := queryParams.Get("corpcode")
	if openid == "" || corpcode == "" {
		return nil, xerr.WithCode(xerr.ErrUnknown, fmt.Sprintf("openid or corpcode is empty, UID: %v", req.UID))
	}

	var yxyResp GetBusAuthTokenYxyResp
	r, err = yxyClient.HttpSendPost(consts.GET_BUS_AUTH_TOKEN_URL,
		map[string]interface{}{
			"openid":   openid,
			"corpcode": corpcode,
		}, yxyHeaders, &yxyResp)
	if err != nil {
		return nil, err
	}

	if r.StatusCode() != 200 {
		return nil, xerr.WithCode(xerr.ErrUnknown, fmt.Sprintf("yxy response: %v", r))
	}

	return &types.GetBusAuthResp{
		Token: yxyResp.Token,
	}, nil
}
