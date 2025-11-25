package main

import (
	"context"
	"flag"
	"yxy-go/internal/config"
	"yxy-go/internal/cron"
	"yxy-go/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
)

var configFile = flag.String("f", "etc/yxy-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	c.MustSetUp()
	logx.DisableStat()

	ctx := svc.NewServiceContext(c)
	cronJob := cron.NewCronJob(context.Background(), ctx)
	cronJob.MustRegister()

	logx.Info("启动定时服务")
	ctx.Cron.Start()
	defer ctx.Cron.Stop()

	select {} // Keep the service running
}
