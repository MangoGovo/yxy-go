package bus

import (
	"context"
	"fmt"
	"strconv"

	"yxy-go/internal/consts"
	"yxy-go/internal/svc"
	"yxy-go/internal/types"
	"yxy-go/internal/utils/yxyClient"
	"yxy-go/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetBusMessageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetBusMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetBusMessageLogic {
	return &GetBusMessageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

type GetBusMessageYxyResp struct {
	Results []types.Message `json:"results"`
}

func (l *GetBusMessageLogic) GetBusMessage(req *types.GetBusMessageReq) (resp *types.GetBusMessageResp, err error) {
	var yxyResp GetBusMessageYxyResp
	var errResp yxyClient.YxyBusErrorResp
	client := yxyClient.GetClient()
	r, err := client.R().
		SetQueryParams(map[string]string{
			"page":      strconv.Itoa(req.Page),
			"page_size": strconv.Itoa(req.PageSize),
			"num_pages": "0",
		}).
		SetHeader("Authorization", req.Token).
		SetResult(&yxyResp).
		SetError(&errResp).
		Get(consts.GET_BUS_MESSAGE_URL)
	if err != nil {
		return nil, xerr.WithCode(xerr.ErrHttpClient, err.Error())
	}

	if r.StatusCode() != 200 {
		errCode := xerr.ErrUnknown
		if errResp.Detail.Code == "AUTH_FAIL" {
			errCode = xerr.ErrBusTokenInvalid
		}
		return nil, xerr.WithCode(errCode, fmt.Sprintf("yxy response: %v", r))
	}

	return &types.GetBusMessageResp{
		List: yxyResp.Results,
	}, nil
}
