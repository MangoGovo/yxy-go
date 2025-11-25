package cron

import (
	"flag"
	"testing"
	"yxy-go/internal/config"
	"yxy-go/internal/logic/bus"
	"yxy-go/internal/manager/auth"

	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/conf"
)

var configFile = flag.String("f", "../../etc/yxy-api.yaml", "the config file")

func TestUpdateBusInfo(t *testing.T) {
	flag.Parse()
	var c config.Config
	conf.MustLoad(*configFile, &c)
	uid := c.BusService.UID
	l := &bus.UpdateBusInfoLogic{}
	m := &auth.BusAuthManager{}
	token, err := m.FetchAuthToken(uid)
	assert.NoError(t, err)
	info, err := l.FetchBusInfo(token)
	assert.NoError(t, err)
	assert.NotNil(t, info)
	t.Log(info)
}
