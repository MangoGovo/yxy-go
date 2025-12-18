package cron

import (
	"context"
	"flag"
	"testing"
	"yxy-go/internal/config"
	"yxy-go/internal/logic/bus"
	"yxy-go/internal/manager/auth"
	"yxy-go/internal/svc"

	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/conf"
)

func TestFetchBusAnnouncement(t *testing.T) {
	flag.Parse()
	var c config.Config
	conf.MustLoad(*configFile, &c)
	uid := c.BusService.UID
	l := &bus.GetBusAnnouncementLogic{}
	m := &auth.BusAuthManager{}
	token, err := m.FetchAuthToken(uid)
	assert.NoError(t, err)
	info, err := l.FetchAnnouncement(token)
	assert.NoError(t, err)
	assert.NotNil(t, info)
	t.Log(info)
}

func TestUpdateBusAnnouncement(t *testing.T) {
	flag.Parse()
	var c config.Config
	conf.MustLoad(*configFile, &c)
	c.MustSetUp()
	svcCtx := svc.NewServiceContext(c)
	l := bus.NewGetBusAnnouncementLogic(context.Background(), svcCtx)
	l.UpdateAnnouncement()
}
