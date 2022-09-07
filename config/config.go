package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

const defaultConfigFile = "config.yml"

type Config struct {
	Pools    []Pool   `yaml:"pools"`
	DB       DB       `yaml:"db"`
	Telegram Telegram `yaml:"telegram"`
}

type DB struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Dbname   string `yaml:"dbname"`
}

type Pool struct {
	ID   string `yaml:"id"`
	Type string `yaml:"type"`
	RPC  string `yaml:"rpc"`
}

type Telegram struct {
	Token string `yaml:"token"`
	Chat  int64  `yaml:"chat"`
}

func Load() (*Config, error) {
	data, err := os.ReadFile(defaultConfigFile)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
