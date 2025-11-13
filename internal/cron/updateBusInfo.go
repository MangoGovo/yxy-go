package cron

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"yxy-go/internal/consts"
	"yxy-go/internal/logic/bus"
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
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateBusInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateBusInfoLogic {
	return &UpdateBusInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateBusInfoLogic) UpdateBusInfoLogic() {
	l.Logger.Info("Start to update bus info at ", time.Now().Format("2006-01-02 15:04:05"))
	err := l.updateBusInfo()
	retries := 0
	maxRetries := l.svcCtx.Config.BusService.MaxRetries

	for err != nil && retries < maxRetries {
		l.Logger.Errorf("Update bus info failed, retrying... (attempt %d/%d): %v", retries+1, maxRetries, err)
		time.Sleep(time.Second * 5) // Wait 5 seconds between retries
		err = l.updateBusInfo()
		retries++
	}

	if err != nil {
		l.Logger.Errorf("Update bus info failed after %d retries: %v", maxRetries, err)
	} else {
		l.Logger.Info("Update bus info success at ", time.Now().Format("2006-01-02 15:04:05"))
	}
}

func (l *UpdateBusInfoLogic) updateBusInfo() error {
	var busData []types.BusInfo
	token, err := l.getBusAuthToken(l.svcCtx.Config.BusService.UID)
	if err != nil {
		l.Logger.Errorf("Get bus auth token failed: %v", err)
		return err
	}

	busInfoList, err := l.fetchBusInfo(token)

	if err != nil {
		l.Logger.Errorf("Fetch bus info failed: %v", err)
		return err
	}

	for _, busInfo := range busInfoList.Results {
		var tmp types.BusInfo
		tmp.ID = busInfo.ID
		tmp.Name = busInfo.Name
		tmp.Price = busInfo.Price

		for _, station := range busInfo.Station {
			tmp.Stations = append(tmp.Stations, types.BusStation{
				ID:   station.ID,
				Name: station.Name,
				Seq:  station.Order,
			})
		}

		busTimeResp, err := l.fetchBusTime(token, busInfo.ID)
		if err != nil {
			l.Logger.Errorf("Fetch bus time failed: %v", err)
			continue
		}

		for _, busTime := range busTimeResp {
			busDataResp, err := l.fetchBusDate(token, busInfo.ID, busTime.ID)
			if err != nil {
				l.Logger.Errorf("Fetch bus date failed: %v", err)
				continue
			}

			if len(busDataResp.Results) == 0 {
				tmp.BusTime = append(tmp.BusTime, types.BusTime{
					DepartureTime: busTime.DepartureTime,
					ID:            busTime.ID,
				})
			} else {
				tmp.BusTime = append(tmp.BusTime, types.BusTime{
					DepartureTime: busTime.DepartureTime,
					ID:            busTime.ID,
					RemainSeats:   busDataResp.Results[0].RemainSeats,
					OrderedSeats:  busDataResp.Results[0].OrderedSeats,
				})
			}
		}
		busData = append(busData, tmp)
	}

	err = l.svcCtx.Rdb.Del(l.ctx, "BusInfo").Err()
	if err != nil {
		l.Logger.Errorf("Delete bus info failed: %v", err)
		return err
	}

	for _, bus := range busData {
		data, err := jsonx.Marshal(bus)
		if err != nil {
			l.Logger.Errorf("Marshal bus info failed: %v", err)
			return err
		}
		err = l.svcCtx.Rdb.RPush(l.ctx, "BusInfo", data).Err()
		if err != nil {
			l.Logger.Errorf("Push bus info failed: %v", err)
			return err
		}
	}

	return nil
}

func (l *UpdateBusInfoLogic) getBusAuthToken(yxyUID string) (string, error) {
	cacheKey := "bus:auth_token:" + yxyUID
	cachedToken, err := l.svcCtx.Rdb.Get(l.ctx, cacheKey).Result()
	if errors.Is(err, redis.Nil) {
		authLogic := bus.NewGetBusAuthLogic(l.ctx, l.svcCtx)
		resp, err := authLogic.GetBusAuth(&types.GetBusAuthReq{
			UID: yxyUID,
		})
		if err != nil {
			return "", err
		}
		if err := l.svcCtx.Rdb.Set(l.ctx, cacheKey, resp.Token, 7*24*time.Hour).Err(); err != nil {
			return "", err
		}
		return resp.Token, nil
	} else if err != nil {
		return "", err
	}
	return cachedToken, nil
}

func (l *UpdateBusInfoLogic) fetchBusInfo(token string) (GetBusInfoYxyResp, error) {
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
		return GetBusInfoYxyResp{}, xerr.WithCode(xerr.ErrHttpClient, err.Error())
	}

	if r.StatusCode() == 400 {
		return GetBusInfoYxyResp{}, xerr.WithCode(xerr.ErrHttpClient, fmt.Sprintf("yxy response: %v", r))
	} else if r.StatusCode() == 500 {
		return GetBusInfoYxyResp{}, xerr.WithCode(xerr.ErrHttpClient, fmt.Sprintf("yxy response: %v", r))
	}

	// fmt.Println(yxyResp)
	return yxyResp, nil
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
