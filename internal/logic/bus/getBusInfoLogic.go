package bus

import (
	"context"
	"encoding/json"
	"strings"
	"time"
	"yxy-go/internal/consts"
	"yxy-go/internal/manager/auth"
	"yxy-go/internal/svc"
	"yxy-go/internal/types"
	"yxy-go/internal/utils/yxyClient"
	"yxy-go/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetBusInfoLogic struct {
	logx.Logger
	ctx         context.Context
	svcCtx      *svc.ServiceContext
	authManager *auth.BusAuthManager
}

func NewGetBusInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetBusInfoLogic {
	return &GetBusInfoLogic{
		authManager: auth.NewBusAuthManager(ctx, svcCtx),
		Logger:      logx.WithContext(ctx),
		ctx:         ctx,
		svcCtx:      svcCtx,
	}
}

func (l *GetBusInfoLogic) GetBusInfo(req *types.GetBusInfoReq) (*types.GetBusInfoResp, error) {
	if req.Search == "" {
		// 全量获取
		return l.getBusInfoFromCache(func(_ types.BusInfo) bool {
			return true
		})
	}
	uid := l.svcCtx.Config.BusService.UID
	resp, err := l.authManager.WithAuthToken(uid, func(token string) (any, error) {
		return l.SearchBusInfo(token, req.Search)
	})
	if err != nil {
		return nil, err
	}
	busData, ok := resp.(*types.GetBusInfoResp)
	if !ok {
		return nil, xerr.WithCode(xerr.ErrUnknown, "解析校车信息失败")
	}
	return busData, nil
}

func (l *GetBusInfoLogic) getBusInfoFromCache(filter func(types.BusInfo) bool) (*types.GetBusInfoResp, error) {
	// 全量获取校车信息
	busInfoListRaw, err := l.svcCtx.Rdb.LRange(l.ctx, "bus:info:data", 0, -1).Result()
	if err != nil {
		return nil, err
	}
	busInfoList := make([]types.BusInfo, 0)
	for _, raw := range busInfoListRaw {
		var tmp types.BusInfo
		if err = json.Unmarshal([]byte(raw), &tmp); err != nil {
			return nil, err
		}
		if filter(tmp) {
			busInfoList = append(busInfoList, tmp)
		}
	}

	// 获取更新时间
	updatedAt, err := l.svcCtx.Rdb.Get(l.ctx, "bus:info:updated_at").Int64()
	if err != nil {
		return nil, err
	}
	return &types.GetBusInfoResp{
		UpdatedAt: time.UnixMilli(updatedAt).Format("2006-01-02 15:04:05"),
		List:      busInfoList,
	}, nil
}

// FetchAllBusInfo 获取全量校车信息
func (l *GetBusInfoLogic) FetchAllBusInfo(token string) ([]types.BusInfo, error) {
	busInfoListRaw, err := l.fetchBusList(token, "")
	if err != nil {
		l.Logger.Errorf("获取校车信息失败, http 请求失败")
		return nil, err
	}
	busInfoList := make([]types.BusInfo, len(busInfoListRaw.Results))
	for i := range busInfoList {
		// 填充字段
		raw := busInfoListRaw.Results[i]
		info := types.BusInfo{
			ID:       raw.ID,
			Name:     raw.Name,
			Price:    raw.Price,
			Stations: make([]string, len(raw.Station)),
		}
		for j, station := range raw.Station {
			info.Stations[j] = station.Name
		}
		// 获取班次
		busScheduleRespRaw, err := l.fetchBusSchedule(token, raw.ID)
		if err != nil {
			l.Logger.Errorf("获取校车时间失败, %v", err)
			return nil, err
		}
		info.BusTime = make([]types.BusTime, 0, len(busScheduleRespRaw))

		for _, busTime := range busScheduleRespRaw {
			// 获取各个班次预约情况
			ordered, remaining, departureDatetime, err := l.fetchBusReservation(token, info.ID, busTime.ID)
			if err != nil {
				l.Logger.Errorf("获取校车日期失败, %v", err)
				continue
			}
			if ordered == 0 && remaining == 0 {
				continue
			}
			info.BusTime = append(info.BusTime, types.BusTime{
				DepartureTime: departureDatetime,
				RemainSeats:   remaining,
				OrderedSeats:  ordered,
			})
		}
		busInfoList[i] = info
	}
	return busInfoList, nil
}

func (l *GetBusInfoLogic) SearchBusInfo(token, search string) (*types.GetBusInfoResp, error) {
	busInfoListRaw, err := l.fetchBusList(token, search)
	if err != nil {
		l.Logger.Errorf("获取校车信息失败, http 请求失败")
		return nil, err
	}
	// 存储所有BusID的集合
	IDSet := make(map[string]struct{})
	for _, raw := range busInfoListRaw.Results {
		IDSet[raw.ID] = struct{}{}
	}

	return l.getBusInfoFromCache(func(item types.BusInfo) bool {
		_, exists := IDSet[item.ID]
		return exists
	})
}

// List:M -> Schedule:N -> Reservation: O(M*N)
// fetchBusList 获取校车信息列表
func (l *GetBusInfoLogic) fetchBusList(token, search string) (*FetchBusInfoYxyResp, error) {
	var yxyResp FetchBusInfoYxyResp

	client := yxyClient.GetClient()
	_, err := client.R().
		SetQueryParams(map[string]string{
			"search":    search,
			"page":      "1",
			"page_size": "999",
		}).
		SetHeader("Authorization", token).
		SetResult(&yxyResp).
		Get(consts.GET_BUS_INFO_URL)

	if err != nil {
		l.Logger.Errorf("Error sending request to %s: %v\n", consts.GET_BUS_INFO_URL, err)
		return nil, xerr.WithCode(xerr.ErrHttpClient, err.Error())
	}
	return &yxyResp, nil
}

// fetchBusSchedule 获取校车班次
func (l *GetBusInfoLogic) fetchBusSchedule(token, busID string) ([]FetchBusScheduleYxyResp, error) {
	// busTime 接口返回的是一个列表，每一项中的 departure_time 才是有效的班车时间，而不是busTime中的项
	var yxyResp []FetchBusScheduleYxyResp

	// url := fmt.Sprintf(consts.GET_BUS_TIME_URL, busID)
	url := strings.Replace(consts.GET_BUS_TIME_URL, "{id}", busID, 1)

	client := yxyClient.GetClient()

	_, err := client.R().
		SetQueryParams(map[string]string{
			"shuttle_type": "-10",
		}).
		SetHeader("Authorization", token).
		SetResult(&yxyResp).
		Get(url)

	if err != nil {
		l.Logger.Errorf("Error sending request to %s: %v\n", consts.GET_BUS_TIME_URL, err)
		return nil, xerr.WithCode(xerr.ErrHttpClient, err.Error())
	}

	return yxyResp, nil
}

// fetchBusReservation 获取校车班次预约情况
func (l *GetBusInfoLogic) fetchBusReservation(token, busID, busScheduleID string) (ordered, remaining int, departureDatetime string, err error) {
	var yxyResp FetchBusReservationYxyResp
	ordered, remaining = 0, 0
	url := strings.Replace(consts.GET_BUS_DATE_URL, "{id}", busID, 1)

	client := yxyClient.GetClient()

	_, err = client.R().
		SetQueryParams(map[string]string{
			"shuttle_bus_time": busScheduleID,
		}).
		SetHeader("Authorization", token).
		SetResult(&yxyResp).
		Get(url)
	if err != nil {
		l.Logger.Errorf("获取校车班次预约情况失败, Http请求失败  %s: %v", consts.GET_BUS_DATE_URL, err)
		return 0, 0, "", xerr.WithCode(xerr.ErrHttpClient, err.Error())
	}
	result := yxyResp.Results
	if len(result) == 0 {
		return 0, 0, "", nil
	}
	return result[0].OrderedSeats, result[0].RemainSeats, result[0].DepartureDatetime, nil
}
