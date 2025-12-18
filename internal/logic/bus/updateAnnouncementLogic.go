package bus

import (
	"time"
	"yxy-go/internal/types"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/jsonx"
)

// UpdateAnnouncement 获取校车公告信息并重试
func (l *GetBusAnnouncementLogic) UpdateAnnouncement() {
	maxRetries := l.svcCtx.Config.BusService.MaxRetries
	uid := l.svcCtx.Config.BusService.UID
	retries := 0
	var announcementData []types.BusAnnouncement
	for ; retries < maxRetries; retries++ {
		resp, err := l.authManager.WithAuthToken(uid, func(token string) (any, error) {
			return l.FetchAnnouncement(token)
		})
		_announcementData, ok := resp.([]types.BusAnnouncement)
		if err == nil && ok {
			l.Logger.Info("成功获取校车公告信息")
			announcementData = _announcementData
			break
		}
		l.Logger.Errorf("获取校车公告信息失败, 重试中... (重试次数 %d/%d): %v", retries+1, maxRetries, err)
		time.Sleep(time.Second * 5)
	}
	if retries == maxRetries {
		l.Logger.Errorf("获取校车公告信息失败! (总重试次数: %d)", maxRetries)
		return
	}
	if err := l.refreshAnnouncementCache(announcementData); err != nil {
		l.Logger.Errorf("刷新校车公告信息缓存失败: %v", err)
	}
}

// refreshAnnouncementCache 刷新缓存, 采用RPush临时key再Rename的方式保证原子性
func (l *GetBusAnnouncementLogic) refreshAnnouncementCache(announcementData []types.BusAnnouncement) error {
	cacheKey := "bus:announcement:data"
	cacheUpdatedAtKey := "bus:announcement:updated_at"
	tempCacheKey := "bus:announcement:temp_data"
	var pushData []interface{}
	for _, announcement := range announcementData {
		data, err := jsonx.Marshal(announcement)
		if err != nil {
			l.Logger.Errorf("校车公告信息序列化失败: %v", err)
			return err
		}
		pushData = append(pushData, data)
	}
	if len(pushData) == 0 {
		l.Logger.Info("校车公告信息为空，未更新缓存")
		return nil
	}
	_, err := l.svcCtx.Rdb.Pipelined(l.ctx, func(pipe redis.Pipeliner) error {
		pipe.RPush(l.ctx, tempCacheKey, pushData...)
		pipe.Expire(l.ctx, tempCacheKey, time.Second*60)
		return nil
	})
	if err != nil {
		l.Logger.Errorf("更新校车公告信息缓存失败: %v", err)
		return err
	}
	if err = l.svcCtx.Rdb.Rename(l.ctx, tempCacheKey, cacheKey).Err(); err != nil {
		l.Logger.Errorf("更新校车公告信息缓存失败: %v", err)
		l.svcCtx.Rdb.Del(l.ctx, tempCacheKey)
		return err
	}
	l.svcCtx.Rdb.Set(l.ctx, cacheUpdatedAtKey, time.Now().UnixMilli(), 0)
	l.svcCtx.Rdb.Persist(l.ctx, cacheKey)
	return nil
}
