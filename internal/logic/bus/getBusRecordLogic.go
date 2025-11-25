package bus

import (
	"context"
	"fmt"
	"strconv"
	"yxy-go/internal/manager/auth"

	"yxy-go/internal/consts"
	"yxy-go/internal/svc"
	"yxy-go/internal/types"
	"yxy-go/internal/utils/yxyClient"
	"yxy-go/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetBusRecordLogic struct {
	logx.Logger
	ctx         context.Context
	svcCtx      *svc.ServiceContext
	authManager *auth.BusAuthManager
}

func NewGetBusRecordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetBusRecordLogic {
	return &GetBusRecordLogic{
		Logger:      logx.WithContext(ctx),
		ctx:         ctx,
		svcCtx:      svcCtx,
		authManager: auth.NewBusAuthManager(ctx, svcCtx),
	}
}

type GetBusRecordYxyResp struct {
	Results []struct {
		DateInfo struct {
			Info struct {
				ID   string `json:"id"`
				Name string `json:"shuttle_name"`
			} `json:"shuttle_bus_vo"`
		} `json:"shuttle_bus_date_vo"`
		DepartureTime string `json:"departure_datetime"`
		PayTime       string `json:"pay_time"`
	} `json:"results"`
}

func (l *GetBusRecordLogic) GetBusRecord(req *types.GetBusRecordReq) (*types.GetBusRecordResp, error) {
	resp, err := l.authManager.WithAuthToken(req.Uid, func(token string) (any, error) {
		return fetchBusRecord(token, req.Page, req.PageSize, "30")
	})
	if err != nil {
		return nil, err
	}
	yxyResp, ok := resp.(*GetBusRecordYxyResp)
	if !ok {
		return nil, xerr.WithCode(xerr.ErrUnknown, "获取校车记录失败")
	}
	records := make([]types.BusRecord, 0)
	for _, row := range yxyResp.Results {
		record := types.BusRecord{
			ID:            row.DateInfo.Info.ID,
			Name:          row.DateInfo.Info.Name,
			DepartureTime: row.DepartureTime,
			PayTime:       row.PayTime,
		}
		records = append(records, record)
	}

	return &types.GetBusRecordResp{
		List: records,
	}, nil
}

func fetchBusRecord(token string, page int, pageSize int, status string) (yxyResp *GetBusRecordYxyResp, err error) {
	var errResp yxyClient.YxyBusErrorResp
	client := yxyClient.GetClient()
	r, err := client.R().
		SetQueryParams(map[string]string{
			"page":      strconv.Itoa(page),
			"page_size": strconv.Itoa(pageSize),
			"status":    status,
		}).
		SetHeader("Authorization", token).
		SetResult(&yxyResp).
		SetError(&errResp).
		Get(consts.GET_BUS_RECORD_URL)
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
	return yxyResp, nil
}
