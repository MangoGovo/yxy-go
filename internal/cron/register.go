package cron

import (
	"context"
	"yxy-go/internal/logic/bus"
	"yxy-go/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CronJob struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCronJob(ctx context.Context, svcCtx *svc.ServiceContext) *CronJob {
	return &CronJob{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (c *CronJob) MustRegister() {
	_, err := c.svcCtx.Cron.AddFunc(c.svcCtx.Config.LowBattery.CronTime, func() {
		l := NewSendLowBatteryAlertLogic(c.ctx, c.svcCtx)
		l.Logger.Info("开始发送低电量提醒")
		l.SendLowBatteryAlertLogic()
		l.Logger.Info("结束发送低电量提醒")
	})
	if err != nil {
		panic(err)
	}
	c.Logger.Info("低电量提醒定时任务注册成功")

	_, err = c.svcCtx.Cron.AddFunc(c.svcCtx.Config.BusService.BusInfoCronTime, func() {
		l := bus.NewGetBusInfoLogic(c.ctx, c.svcCtx)
		l.Logger.Info("开始获取校车信息")
		l.UpdateBusInfo()
		l.Logger.Info("结束获取校车信息")
	})
	if err != nil {
		panic(err)
	}
	c.Logger.Info("校车信息查询定时任务注册成功")

	_, err = c.svcCtx.Cron.AddFunc(c.svcCtx.Config.BusService.BusAnnouncementCronTime, func() {
		l := bus.NewGetBusAnnouncementLogic(c.ctx, c.svcCtx)
		l.Logger.Info("开始获取校车公告信息")
		l.UpdateAnnouncement()
		l.Logger.Info("结束获取校车公告信息")
	})
	c.Logger.Info("校车公告信息查询定时任务注册成功")
	if err != nil {
		panic(err)
	}
}
