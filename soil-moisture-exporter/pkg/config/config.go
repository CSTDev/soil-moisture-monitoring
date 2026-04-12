package config

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"

	yaml "gopkg.in/yaml.v3"
)

type ConfigData struct {
	PollIntervalSeconds int    `yaml:"pollIntervalSeconds"`
	DatabaseFile        string `yaml:"databaseFile"`
	Health              struct {
		Namespace        string `yaml:"namespace"`
		AppName          string `yaml:"appName"`
		HearbeatInterval int    `yaml:"heartBeatInterval"`
	} `yaml:"health"`
	MQTT struct {
		Broker   string
		Port     int
		Protocol string
		QOS      byte `yaml:"qos"`
		Username string
		Password string
	} `yaml:"mqtt"`
}

func ReadConfig(fileName string) (ConfigData, error) {
	buf, err := os.ReadFile(fileName)
	if err != nil {
		return ConfigData{}, err
	}

	c := &ConfigData{}
	err = yaml.Unmarshal(buf, c)
	if err != nil {
		return ConfigData{}, fmt.Errorf("in file %q: %w", fileName, err)
	}
	if c.PollIntervalSeconds == 0 {
		c.PollIntervalSeconds = 60
	}

	log.Printf("Config: \n %#v \n", *c)
	return *c, nil
}
