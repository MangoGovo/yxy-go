package bus

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
	"yxy-go/internal/consts"
	"yxy-go/internal/manager/auth"
	"yxy-go/internal/svc"
	"yxy-go/internal/types"
	"yxy-go/internal/utils/yxyClient"
	"yxy-go/pkg/xerr"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetBusInfoYxyResp struct {
	Count   int `json:"count"`
	Results []struct {
		ID      string `json:"id"`
		Name    string `json:"shuttle_name"`
		Price   int    `json:"price"`
		Station []struct {
			ID    string `json:"id"`
			Name  string `json:"station_name"`
			Order int    `json:"station_seq"`
		} `json:"go_stations_json"`
	} `json:"results"`
}

type GetBusTimeYxyResp struct {
	Info struct {
		Name string `json:"shuttle_name"`
	} `json:"shuttle_bus_vo"`
	ID            string `json:"id"`
	DepartureTime string `json:"departure_time"`
}

type GetBusDateYxyResp struct {
	// 这里看似是一个列表但是他只会返回一个...
	Results []struct {
		OrderedSeats int `json:"order_cnt"`
		RemainSeats  int `json:"remaining_seats"`
	} `json:"results"`
}

type UpdateBusInfoLogic struct {
	logx.Logger
	ctx         context.Context
	svcCtx      *svc.ServiceContext
	authManager *auth.BusAuthManager
}

func NewUpdateBusInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateBusInfoLogic {
	return &UpdateBusInfoLogic{
		Logger:      logx.WithContext(ctx),
		ctx:         ctx,
		svcCtx:      svcCtx,
		authManager: auth.NewBusAuthManager(ctx, svcCtx),
	}
}

// UpdateBusInfoLogic 获取校车信息并重试
func (l *UpdateBusInfoLogic) UpdateBusInfoLogic() {
	maxRetries := l.svcCtx.Config.BusService.MaxRetries
	uid := l.svcCtx.Config.BusService.UID
	retries := 0
	var busData []*types.BusInfo
	for ; retries < maxRetries; retries++ {
		resp, err := l.authManager.WithAuthToken(uid, func(token string) (any, error) {
			return l.FetchBusInfo(token)
		})
		_busData, ok := resp.([]*types.BusInfo)
		if err == nil && ok {
			l.Logger.Info("成功获取校车信息")
			busData = _busData
			break
		}
		l.Logger.Errorf("获取校车信息失败, 重试中... (重试次数 %d/%d): %v", retries+1, maxRetries, err)
		time.Sleep(time.Second * 5)
	}
	if retries == maxRetries {
		l.Logger.Errorf("获取校车信息失败! (总重试次数: %s)", maxRetries)
		return
	}
	if err := l.refreshCache(busData); err != nil {
		l.Logger.Errorf("刷新校车信息缓存失败: %v", err)
	}
}

func (l *UpdateBusInfoLogic) FetchBusInfo(token string) ([]*types.BusInfo, error) {
	var busData []*types.BusInfo
	busInfoList, err := l.fetchBusBaseInfo(token)
	if err != nil {
		l.Logger.Errorf("获取校车信息失败, http 请求失败")
		return nil, err
	}
	for _, busInfo := range busInfoList.Results {
		var tmp types.BusInfo
		tmp.ID = busInfo.ID
		tmp.Name = busInfo.Name
		tmp.Price = busInfo.Price

		for _, station := range busInfo.Station {
			tmp.Stations = append(tmp.Stations, station.Name)
		}
		busTimeResp, err := l.fetchBusTime(token, busInfo.ID)
		if err != nil {
			l.Logger.Errorf("获取校车时间失败, %v", err)
			return nil, err
		}
		for _, busTime := range busTimeResp {
			busDataResp, err := l.fetchBusDate(token, busInfo.ID, busTime.ID)
			if err != nil {
				l.Logger.Errorf("获取校车日期失败, %v", err)
				continue
			}

			if len(busDataResp.Results) == 0 {
				tmp.BusTime = append(tmp.BusTime, types.BusTime{
					DepartureTime: busTime.DepartureTime,
				})
			} else {
				tmp.BusTime = append(tmp.BusTime, types.BusTime{
					DepartureTime: busTime.DepartureTime,
					RemainSeats:   busDataResp.Results[0].RemainSeats,
					OrderedSeats:  busDataResp.Results[0].OrderedSeats,
				})
			}
		}
		busData = append(busData, &tmp)
	}
	return busData, nil
}

// refreshCache 刷新缓存, 采用RPush临时key再Rename的方式保证原子性
func (l *UpdateBusInfoLogic) refreshCache(busData []*types.BusInfo) error {
	cacheKey := "bus:info:data"
	cacheUpdatedAtKey := "bus:info:updated_at"
	tempCacheKey := "bus:info:temp_data"
	var pushData []interface{}
	for _, bus := range busData {
		data, err := jsonx.Marshal(bus)
		if err != nil {
			l.Logger.Errorf("校车信息反序列化失败: %v", err)
			return err
		}
		pushData = append(pushData, data)
	}
	if len(pushData) == 0 {
		return xerr.WithCode(xerr.ErrUnknown, "校车信息为空，未更新缓存")
	}
	_, err := l.svcCtx.Rdb.Pipelined(l.ctx, func(pipe redis.Pipeliner) error {
		pipe.RPush(l.ctx, tempCacheKey, pushData...)
		pipe.Expire(l.ctx, tempCacheKey, time.Second*60)
		return nil
	})
	if err != nil {
		l.Logger.Errorf("更新校车信息缓存失败: %v", err)
		return err
	}
	if err = l.svcCtx.Rdb.Rename(l.ctx, tempCacheKey, cacheKey).Err(); err != nil {
		l.Logger.Errorf("更新校车信息缓存失败: %v", err)
		l.svcCtx.Rdb.Del(l.ctx, tempCacheKey)
		return err
	}
	l.svcCtx.Rdb.Set(l.ctx, cacheUpdatedAtKey, time.Now().UnixMilli(), 0)
	l.svcCtx.Rdb.Persist(l.ctx, cacheKey)
	return nil
}

func (l *UpdateBusInfoLogic) fetchBusBaseInfo(token string) (*GetBusInfoYxyResp, error) {
	var yxyResp GetBusInfoYxyResp

	client := yxyClient.GetClient()
	r, err := client.R().
		SetQueryParams(map[string]string{
			"page":      "1",
			"page_size": "999",
		}).
		SetHeader("Authorization", token).
		SetResult(&yxyResp).
		Get(consts.GET_BUS_INFO_URL)

	if err != nil {
		log.Printf("Error sending request to %s: %v\n", consts.GET_BUS_INFO_URL, err)
		return nil, xerr.WithCode(xerr.ErrHttpClient, err.Error())
	}

	if r.StatusCode() == 400 {
		return nil, xerr.WithCode(xerr.ErrHttpClient, fmt.Sprintf("yxy response: %v", r))
	} else if r.StatusCode() == 500 {
		return nil, xerr.WithCode(xerr.ErrHttpClient, fmt.Sprintf("yxy response: %v", r))
	}

	return &yxyResp, nil
}

func (l *UpdateBusInfoLogic) fetchBusTime(token, busID string) ([]GetBusTimeYxyResp, error) {
	// bustime 接口返回的是一个列表，每一项中的 departure_time 才是有效的班车时间，而不是bustime中的项
	var yxyResp []GetBusTimeYxyResp

	// url := fmt.Sprintf(consts.GET_BUS_TIME_URL, busID)
	url := strings.Replace(consts.GET_BUS_TIME_URL, "{id}", busID, 1)

	client := yxyClient.GetClient()

	r, err := client.R().
		SetQueryParams(map[string]string{
			"shuttle_type": "-10",
		}).
		SetHeader("Authorization", token).
		SetResult(&yxyResp).
		Get(url)

	if err != nil {
		log.Printf("Error sending request to %s: %v\n", consts.GET_BUS_TIME_URL, err)
		return nil, xerr.WithCode(xerr.ErrHttpClient, err.Error())
	}

	if r.StatusCode() == 400 {
		return nil, xerr.WithCode(xerr.ErrHttpClient, fmt.Sprintf("yxy response: %v", r))
	} else if r.StatusCode() == 500 {
		return nil, xerr.WithCode(xerr.ErrHttpClient, fmt.Sprintf("yxy response: %v", r))
	}

	return yxyResp, nil
}

func (l *UpdateBusInfoLogic) fetchBusDate(token, busID, busTimeID string) (GetBusDateYxyResp, error) {
	var yxyResp GetBusDateYxyResp

	url := strings.Replace(consts.GET_BUS_DATE_URL, "{id}", busID, 1)

	client := yxyClient.GetClient()

	r, err := client.R().
		SetQueryParams(map[string]string{
			"shuttle_bus_time": busTimeID,
		}).
		SetHeader("Authorization", token).
		SetResult(&yxyResp).
		Get(url)

	if err != nil {
		log.Printf("Error sending request to %s: %v\n", consts.GET_BUS_DATE_URL, err)
		return GetBusDateYxyResp{}, xerr.WithCode(xerr.ErrHttpClient, err.Error())
	}

	if r.StatusCode() == 400 {
		return GetBusDateYxyResp{}, xerr.WithCode(xerr.ErrHttpClient, fmt.Sprintf("yxy response: %v", r))
	} else if r.StatusCode() == 500 {
		return GetBusDateYxyResp{}, xerr.WithCode(xerr.ErrHttpClient, fmt.Sprintf("yxy response: %v", r))
	}

	return yxyResp, nil
}
