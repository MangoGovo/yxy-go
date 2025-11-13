package config

import (
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	EnableCron bool
	CronTime   string
	Mysql      struct {
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
	BusService struct {
		UID        string
		MaxRetries int
		CronTime   string
	}
}
