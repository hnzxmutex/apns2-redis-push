package main

import (
	"flag"
	"github.com/go-ini/ini"
)

type GlobalConfig struct {
	CertificatePath                        string
	RedisAddr, RedisPassword, RedisListKey string
	RedisDB                                int64
	RoutineCount                           uint
	Mode, Topic                            string
}

//get config
func getConfig() GlobalConfig {
	var configPath string
	flag.StringVar(&configPath, "config-path", "./config.ini", "the config.ini path")
	flag.Parse()

	config := GlobalConfig{
		CertificatePath: "./apns.pem",
		RedisAddr:       "127.0.0.1:6379",
		RedisPassword:   "",
		RedisListKey:    "push_list",
		RoutineCount:    8,
		Mode:            "development",
		Topic:           "com.testapp.appname",
	}

	ini.MapTo(&config, configPath)
	return config
}
