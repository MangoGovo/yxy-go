package config

import (
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	Mysql struct {
		Host   string
		Port   int
		User   string
		Pass   string
		DBName string
	}
	Redis struct {
		Host string
		Port int
		Pass string
		DB   int
	}
	LowBattery struct {
		MiniProgram struct {
			AppID        string
			Secret       string
			HttpDebug    bool
			LogLevel     string
			LogInfoFile  string
			LogErrorFile string
			LogStdout    bool
			State        string
			TemplateID   string
		}
		EnableCron bool
		CronTime   string
	}
	BusService struct {
		UID                     string
		MaxRetries              int
		BusInfoCronTime         string
		BusAnnouncementCronTime string
	}
}
