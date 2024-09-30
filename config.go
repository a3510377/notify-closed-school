package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path"
	"time"

	"gopkg.in/yaml.v3"
)

// last update: 2023/07/28
var AreaNameStrings = []string{
	"基隆市",
	"臺北市",
	"新北市",
	"桃園市",
	"新竹市",
	"新竹縣",

	"苗栗縣",
	"臺中市",
	"彰化縣",
	"雲林縣",
	"南投縣",

	"嘉義市",
	"嘉義縣",
	"臺南市",
	"高雄市",
	"屏東縣",

	"宜蘭縣",
	"花蓮縣",
	"臺東縣",

	"澎湖縣",
	"連江縣",
	"金門縣",
}

type Config struct {
	Discord   DiscordConfig `yaml:"discord"`
	Line      LineConfig    `yaml:"line"`
	AreaNames []string      `yaml:"area_name"`
}

type DiscordConfig struct {
	Enable     bool     `yaml:"enable"`
	TOKEN      string   `yaml:"TOKEN"`
	Webhooks   []string `yaml:"webhook"`
	ChannelIDs []int64  `yaml:"channel_ids"`
}

type LineConfig struct {
	Enable bool     `yaml:"enable"`
	Tokens []string `yaml:"tokens"`
}

var (
	ConfigData = &Config{}
	StopWatch  = false
)

const (
	ConfigFilePath = "./data/config.yaml"
	TmpFilePath    = "./data/tmp"
)

func init() {
	yamlFile, err := os.ReadFile(ConfigFilePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Fatal(err)
		}
		config := NewConfig()
		data, _ := yaml.Marshal(config)
		os.MkdirAll(path.Dir(ConfigFilePath), 0777)
		os.WriteFile(ConfigFilePath, data, 0777)
	} else {
		yaml.Unmarshal(yamlFile, &ConfigData)
	}

	// watch config file
	go func() {
		for !StopWatch {
			err := watchFile(ConfigFilePath)
			if err != nil {
				log.Println("config watch error", err)
				time.Sleep(time.Second * 5)
			}

			// reload config
			yamlFile, err := os.ReadFile(ConfigFilePath)
			if err != nil {
				log.Println("config watch error", err)
				time.Sleep(time.Second * 5)
			} else {
				yaml.Unmarshal(yamlFile, &ConfigData)
			}
		}
	}()
}

func NewConfig() *Config {
	return &Config{
		Discord:   DiscordConfig{Enable: false},
		Line:      LineConfig{Enable: false},
		AreaNames: AreaNameStrings,
	}
}

// is ease watch file func
func watchFile(filePath string) error {
	initialStat, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	for {
		stat, err := os.Stat(filePath)
		if err != nil {
			return err
		}

		if stat.Size() != initialStat.Size() || stat.ModTime() != initialStat.ModTime() {
			break
		}

		// sleep 1 second
		time.Sleep(time.Second)
	}
	return nil
}

func GetTmpDate() (value map[string]map[string]bool) {
	if data, err := os.ReadFile(TmpFilePath); err == nil {
		json.Unmarshal(data, &value)
		return
	}
	return map[string]map[string]bool{} // if file not exist
}

func WriteTmpDate(v any) (err error) {
	bytes, err := json.Marshal(v)
	if err == nil {
		os.MkdirAll(path.Dir(ConfigFilePath), 0777)
		os.WriteFile(TmpFilePath, bytes, 0777)
	}
	return
}
