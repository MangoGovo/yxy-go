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

var configFile = flag.String("f", "../../etc/yxy-api.yaml", "the config file")

func TestFetchBusInfo(t *testing.T) {
	flag.Parse()
	var c config.Config
	conf.MustLoad(*configFile, &c)
	uid := c.BusService.UID
	l := &bus.GetBusInfoLogic{}
	m := &auth.BusAuthManager{}
	token, err := m.FetchAuthToken(uid)
	assert.NoError(t, err)
	info, err := l.FetchAllBusInfo(token)
	assert.NoError(t, err)
	assert.NotNil(t, info)
	t.Log(info)
}

func TestUpdateBusInfo(t *testing.T) {
	flag.Parse()
	var c config.Config
	conf.MustLoad(*configFile, &c)
	c.MustSetUp()
	svcCtx := svc.NewServiceContext(c)
	l := bus.NewGetBusInfoLogic(context.Background(), svcCtx)
	l.UpdateBusInfo()
}
