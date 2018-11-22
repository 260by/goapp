package config

import (
	"io/ioutil"
	"gopkg.in/yaml.v2"
)

type config struct {
	Logs []Log
}

type Log struct {
	Name string
	Path struct {
		Dir   string
		Files []string
		Ext string
	}
}

// Config 配置信息
var Config = config{}

// Load 载入配置
func Load(data []byte) error {
	return yaml.Unmarshal(data, &Config)
}

// LoadFile 载入配置文件
func LoadFile(file string) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	return Load(data)
}