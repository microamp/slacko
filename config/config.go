package config

import (
	"encoding/json"
	"io/ioutil"
)

type SlackoConfig struct {
	GoPlaygroundHost string `json:"GoPlaygroundHost"`
	BotName          string `json:"BotName"`
	DebugOn          bool   `json:"DebugOn"`
	CacheSize        int    `json:"CacheSize"`
}

func ReadConfig(filename string) (*SlackoConfig, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var config *SlackoConfig
	if err = json.Unmarshal(bytes, &config); err != nil {
		return nil, err
	}
	return config, nil
}
