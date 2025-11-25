package bus

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"
	"yxy-go/internal/svc"
	"yxy-go/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetBusInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetBusInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetBusInfoLogic {
	return &GetBusInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetBusInfoLogic) GetBusInfo(req *types.GetBusInfoReq) (resp *types.GetBusInfoResp, err error) {
	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize - 1

	busInfoList, err := l.svcCtx.Rdb.LRange(l.ctx, "BusInfo", int64(start), int64(end)).Result()
	if err != nil {
		return nil, err
	}
	updatedAtStr, err := l.svcCtx.Rdb.Get(l.ctx, "bus:info:updated_at").Result()
	if err != nil {
		return nil, err
	}
	updatedAt, err := strconv.ParseInt(updatedAtStr, 10, 64)
	if err != nil {
		return nil, err
	}

	filteredBusInfoList := make([]types.BusInfo, 0)
	for _, busInfo := range busInfoList {
		if strings.Contains(busInfo, req.Search) {
			var tmp types.BusInfo
			err := json.Unmarshal([]byte(busInfo), &tmp)
			if err != nil {
				l.Errorf("failed to unmarshal bus info: %v", err)
				continue
			}
			filteredBusInfoList = append(filteredBusInfoList, tmp)
		}
	}

	return &types.GetBusInfoResp{
		UpdatedAt: time.UnixMilli(updatedAt).Format("2006-01-02 15:04:05"),
		List:      filteredBusInfoList,
	}, nil
}
