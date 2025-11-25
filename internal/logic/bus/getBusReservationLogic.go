package bus

import (
	"context"
	"yxy-go/internal/manager/auth"
	"yxy-go/pkg/xerr"

	"yxy-go/internal/svc"
	"yxy-go/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetBusReservationLogic struct {
	logx.Logger
	ctx         context.Context
	svcCtx      *svc.ServiceContext
	authManager *auth.BusAuthManager
}

func NewGetBusReservationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetBusReservationLogic {
	return &GetBusReservationLogic{
		Logger:      logx.WithContext(ctx),
		ctx:         ctx,
		svcCtx:      svcCtx,
		authManager: auth.NewBusAuthManager(ctx, svcCtx),
	}
}

func (l *GetBusReservationLogic) GetBusReservation(req *types.GetBusReservationReq) (*types.GetBusReservationResp, error) {
	resp, err := l.authManager.WithAuthToken(req.Uid, func(token string) (any, error) {
		return fetchBusRecord(token, req.Page, req.PageSize, "20")
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

	return &types.GetBusReservationResp{
		List: records,
	}, nil
}
