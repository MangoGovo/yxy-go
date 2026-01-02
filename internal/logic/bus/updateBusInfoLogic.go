package bus

import (
	"time"
	"yxy-go/internal/types"
	"yxy-go/pkg/xerr"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/jsonx"
)

type FetchBusInfoYxyResp struct {
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

type FetchBusScheduleYxyResp struct {
	Info struct {
		Name string `json:"shuttle_name"`
	} `json:"shuttle_bus_vo"`
	ID            string `json:"id"`
	DepartureTime string `json:"departure_time"`
}

type FetchBusReservationYxyResp struct {
	// 这里看似是一个列表但是他只会返回一个...
	Results []struct {
		OrderedSeats      int    `json:"order_cnt"`
		RemainSeats       int    `json:"remaining_seats"`
		DepartureDatetime string `json:"departure_datetime"`
	} `json:"results"`
}

// UpdateBusInfo 获取校车信息并重试
func (l *GetBusInfoLogic) UpdateBusInfo() {
	maxRetries := l.svcCtx.Config.BusService.MaxRetries
	uid := l.svcCtx.Config.BusService.UID
	retries := 0
	var busData []types.BusInfo
	for ; retries < maxRetries; retries++ {
		resp, err := l.authManager.WithAuthToken(uid, func(token string) (any, error) {
			return l.FetchAllBusInfo(token)
		})
		_busData, ok := resp.([]types.BusInfo)
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

// refreshCache 刷新缓存, 采用RPush临时key再Rename的方式保证原子性
func (l *GetBusInfoLogic) refreshCache(busData []types.BusInfo) error {
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
