package bus

import (
	"context"
	"fmt"

	"yxy-go/internal/consts"
	"yxy-go/internal/svc"
	"yxy-go/internal/types"
	"yxy-go/internal/utils/yxyClient"
	"yxy-go/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetBusQrcodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetBusQrcodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetBusQrcodeLogic {
	return &GetBusQrcodeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

type GetBusQrcodeYxyResp struct {
	Qrcode string `json:"qrcode"`
}

func (l *GetBusQrcodeLogic) GetBusQrcode(req *types.GetBusQrcodeReq) (resp *types.GetBusQrcodeResp, err error) {
	var yxyResp GetBusQrcodeYxyResp
	var errResp yxyClient.YxyBusErrorResp
	client := yxyClient.GetClient()
	r, err := client.R().
		SetHeader("Authorization", req.Token).
		SetResult(&yxyResp).
		SetError(&errResp).
		Get(consts.GET_BUS_QRCODE_URL)
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

	return &types.GetBusQrcodeResp{
		Qrcode: yxyResp.Qrcode,
	}, nil
}
