// --------------------
// File: config.go
// Project: config
// Purpose: 配置处理
// Author: Jan Lam (honos628@foxmail.com)
// Last Modified: 2021-07-30 15:41:27
// --------------------

package config

import (
	"log"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

type ObsAccount struct {
	Endpoint string
	Ak       string
	Sk       string
}

type Config struct {
	Port int
	// ObsAk       string
	// ObsSk       string
	// ObsEndpoint string
	ObsAccounts []ObsAccount
}

var (
	// poc
	defaultPort = 9131
	// defaultEndpoint = "http://19.15.81.222:82"
	// defaultAk       = "FKETSO5GUVN7OSB1SSEQ"
	// defaultSk       = "hCelfb9FNToBsHkiQgYLIaQlVKPpn1xx6d0wiyrt"
	defaultAccounts = []ObsAccount{}
)

func LoadConfig() (c Config) {
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath(".")
	viper.SetEnvPrefix("exporter") // 将自动转为大写
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	viper.BindEnv("port")
	// viper.BindEnv("endpoint")
	// viper.BindEnv("obs.ak")
	// viper.BindEnv("obs.sk")
	viper.SetDefault("port", defaultPort)
	// viper.SetDefault("obs.endpoint", defaultEndpoint)
	// viper.SetDefault("obs.ak", defaultAk)
	// viper.SetDefault("obs.sk", defaultSk)
	viper.SetDefault("obs.accounts", defaultAccounts)
	err := viper.ReadInConfig()
	if err != nil {
		log.Println("read config failed:", err)
		log.Println("use default settings...")
	}
	// 处理port有可能是string类型的问题
	switch viper.Get("port").(type) {
	case int:
		c.Port = viper.Get("port").(int)
	case string:
		c.Port, _ = strconv.Atoi(viper.Get("port").(string))
	}
	// c.ObsEndpoint = viper.Get("obs.endpoint").(string)
	// c.ObsAk = viper.Get("obs.ak").(string)
	// c.ObsSk = viper.Get("obs.sk").(string)
	oa := viper.Get("obs.accounts").([]interface{})
	for _, v := range oa {
		t := v.(map[interface{}]interface{})
		c.ObsAccounts = append(c.ObsAccounts, ObsAccount{
			Endpoint: t["endpoint"].(string),
			Ak:       t["ak"].(string),
			Sk:       t["sk"].(string),
		})
	}
	return
}
