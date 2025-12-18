package bus

import (
	"context"
	"encoding/json"
	"time"
	"yxy-go/internal/consts"
	"yxy-go/internal/manager/auth"
	"yxy-go/internal/svc"
	"yxy-go/internal/types"
	"yxy-go/internal/utils/yxyClient"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetBusAnnouncementLogic struct {
	logx.Logger
	ctx         context.Context
	svcCtx      *svc.ServiceContext
	authManager *auth.BusAuthManager
}

func NewGetBusAnnouncementLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetBusAnnouncementLogic {
	return &GetBusAnnouncementLogic{
		Logger:      logx.WithContext(ctx),
		ctx:         ctx,
		svcCtx:      svcCtx,
		authManager: auth.NewBusAuthManager(ctx, svcCtx),
	}
}

func (l *GetBusAnnouncementLogic) GetBusAnnouncement(req *types.GetBusAnnouncementReq) (resp *types.GetBusAnnouncementResp, err error) {
	page := req.Page
	pageSize := req.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize > 10 || pageSize < 1 {
		pageSize = 10
	}
	return l.getAnnouncementFromCache(page, pageSize)
}

// getAnnouncementFromCache 从缓存获取公告信息
func (l *GetBusAnnouncementLogic) getAnnouncementFromCache(page, pageSize int) (*types.GetBusAnnouncementResp, error) {
	cacheKey := "bus:announcement:data"
	cacheUpdatedAtKey := "bus:announcement:updated_at"

	// 计算分页的起始和结束索引
	start := int64((page - 1) * pageSize)
	end := start + int64(pageSize) - 1

	// 从缓存获取公告列表
	announcementListRaw, err := l.svcCtx.Rdb.LRange(l.ctx, cacheKey, start, end).Result()
	if err != nil {
		return nil, err
	}

	announcementList := make([]types.BusAnnouncement, 0, len(announcementListRaw))
	for _, raw := range announcementListRaw {
		var tmp types.BusAnnouncement
		if err = json.Unmarshal([]byte(raw), &tmp); err != nil {
			return nil, err
		}
		announcementList = append(announcementList, tmp)
	}

	// 获取总数
	total, err := l.svcCtx.Rdb.LLen(l.ctx, cacheKey).Result()
	if err != nil {
		return nil, err
	}

	// 获取更新时间
	updatedAt, err := l.svcCtx.Rdb.Get(l.ctx, cacheUpdatedAtKey).Int64()
	if err != nil {
		return nil, err
	}

	return &types.GetBusAnnouncementResp{
		UpdatedAt: time.UnixMilli(updatedAt).Format("2006-01-02 15:04:05"),
		Total:     total,
		List:      announcementList,
	}, nil
}

type fetchAnnouncementResp struct {
	Result []struct {
		Ctime   string `json:"ctime"`
		Title   string `json:"title"`
		Content string `json:"content"`
		HTML    string `json:"html"`
		Author  string `json:"author"`
	} `json:"results"`
}

func (l *GetBusAnnouncementLogic) FetchAnnouncement(token string) (resp []types.BusAnnouncement, err error) {
	client := yxyClient.GetClient()
	var fetchResp fetchAnnouncementResp
	_, err = client.R().
		SetQueryParams(map[string]string{
			"page_size": "999",
		}).
		SetHeader("Authorization", token).
		SetResult(&fetchResp).
		Get(consts.GET_BUS_ANNOUNCEMENT_URL)
	if err != nil {
		return nil, err
	}
	for _, item := range fetchResp.Result {
		resp = append(resp, types.BusAnnouncement{
			Title:       item.Title,
			Author:      item.Author,
			PublishedAt: item.Ctime,
			Abstract:    item.Content,
			Content:     yxyClient.ParseHTMLAnnouncement(item.HTML),
		})
	}
	return resp, nil
}
