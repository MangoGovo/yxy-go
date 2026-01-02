package svc

import (
	"fmt"
	"time"
	"yxy-go/internal/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/ArtisanCloud/PowerWeChat/v3/src/miniProgram"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"github.com/zeromicro/go-zero/core/logx"
)

type ServiceContext struct {
	Config      config.Config
	DB          *gorm.DB
	Rdb         *redis.Client
	MiniProgram *miniProgram.MiniProgram
	Cron        *cron.Cron
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:      c,
		DB:          NewGorm(c),
		Rdb:         NewRedis(c),
		MiniProgram: NewMiniProgram(c),
		Cron:        NewCron(c),
	}
}

func NewGorm(c config.Config) *gorm.DB {
	if !c.LowBattery.EnableCron {
		return nil
	}
	dsn := fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?charset=utf8mb4&parseTime=True&loc=Local", c.Mysql.User, c.Mysql.Pass, c.Mysql.Host, c.Mysql.Port, c.Mysql.DBName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	return db
}

func NewRedis(c config.Config) *redis.Client {
	if !c.LowBattery.EnableCron {
		return nil
	}
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%v:%v", c.Redis.Host, c.Redis.Port),
		Password: c.Redis.Pass,
		DB:       c.Redis.DB,
	})
}

func NewMiniProgram(c config.Config) *miniProgram.MiniProgram {
	if !c.LowBattery.EnableCron {
		return nil
	}
	mp := c.LowBattery.MiniProgram
	MiniProgramApp, err := miniProgram.NewMiniProgram(&miniProgram.UserConfig{
		AppID:     mp.AppID,
		Secret:    mp.Secret,
		HttpDebug: mp.HttpDebug,
		Log: miniProgram.Log{
			Level:  mp.LogLevel,
			File:   mp.LogInfoFile,
			Error:  mp.LogErrorFile,
			Stdout: mp.LogStdout,
		},
	})
	if err != nil {
		logx.Errorf("MiniProgram init error, err: %v", err)
		panic(err)
	}
	return MiniProgramApp
}

func NewCron(c config.Config) *cron.Cron {
	if !c.LowBattery.EnableCron {
		return nil
	}
	loc, _ := time.LoadLocation("Asia/Shanghai")
	return cron.New(cron.WithLocation(loc))
}
