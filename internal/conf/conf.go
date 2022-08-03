// Copyright 2021 EMQ Technologies Co., Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package conf

import (
	"errors"
	"fmt"
	"github.com/lestrrat-go/file-rotatelogs"
	"github.com/lf-edge/ekuiper/pkg/api"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"os"
	"time"
)

const ConfFileName = "kuiper.yaml"

var (
	Config    *KuiperConf
	IsTesting bool
)

type tlsConf struct {
	Certfile string `yaml:"certfile"`
	Keyfile  string `yaml:"keyfile"`
}

type DomainInfo struct {
	DomainUid   string `yaml:"domain_uid"`
	DomainName  string `yaml:"domain_name"`
	LocalDomain bool   `yaml:"local_domain"`
	Lines       []struct {
		Ipv4       string `yaml:"ipv4"`
		PortConfig []struct {
			Usage string `yaml:"usage"`
			Ports []int  `yaml:"ports"`
		} `yaml:"port_config"`
	} `yaml:"lines"`
}

type KongLocalAddr struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type KongLocal struct {
	Manage      KongLocalAddr `yaml:"manage"`
	Internal    KongLocalAddr `yaml:"internal"`
	External    KongLocalAddr `yaml:"external"`
	KongPlugins []string      `yaml:"plugins"`
}

type KuiperConf struct {
	ServiceUid       string `yaml:"service_uid"`
	ConsulConnection struct {
		Enable         bool   `yaml:"enable"`
		Host           string `yaml:"host"`
		Port           int    `yaml:"port"`
		Datacenter     string `yaml:"datacenter"`
		PropertyKvPath string `yaml:"property_kv_path"`
	} `yaml:"consul_connection"`

	Service struct {
		NodeHost string `yaml:"node_host"`
		DataRoot string `yaml:"data_root"`
	} `yaml:"service"`

	Domains struct {
		Remote string       `yaml:"remote"`
		Local  []DomainInfo `yaml:"local"`
	} `yaml:"domains"`

	Kong struct {
		Remote string    `yaml:"remote"`
		Local  KongLocal `yaml:"local"`
	} `yaml:"kong"`

	Log struct {
		Enable         bool   `yaml:"enable"`
		Level          string `yaml:"level"`
		Mode           string `yaml:"mode"`
		FilesPath      string `yaml:"files_path"`
		RotationCount  int    `yaml:"rotation_count"`
		RotationSize   int    `yaml:"rotation_size"`
		RotationPeriod string `yaml:"rotation_period"`
		Compression    bool   `yaml:"compression"`
	} `yaml:"log"`

	Basic struct {
		Debug          bool     `yaml:"debug"`
		ConsoleLog     bool     `yaml:"consoleLog"`
		FileLog        bool     `yaml:"fileLog"`
		RotateTime     int      `yaml:"rotateTime"`
		MaxAge         int      `yaml:"maxAge"`
		Ip             string   `yaml:"ip"`
		Port           int      `yaml:"port"`
		RestIp         string   `yaml:"restIp"`
		RestPort       int      `yaml:"restPort"`
		RestTls        *tlsConf `yaml:"restTls"`
		Prometheus     bool     `yaml:"prometheus"`
		PrometheusPort int      `yaml:"prometheusPort"`
		PluginHosts    string   `yaml:"pluginHosts"`
		Authentication bool     `yaml:"authentication"`
		IgnoreCase     bool     `yaml:"ignoreCase"`
	} `yaml:"basic"`

	Rule api.RuleOption `yaml:"rule"`

	Sink struct {
		CacheThreshold    int  `yaml:"cacheThreshold"`
		CacheTriggerCount int  `yaml:"cacheTriggerCount"`
		DisableCache      bool `yaml:"disableCache"`
	} `yaml:"sink"`

	Store struct {
		Type  string `yaml:"type"`
		Redis struct {
			Host               string `yaml:"host"`
			Port               int    `yaml:"port"`
			Password           string `yaml:"password"`
			Timeout            int    `yaml:"timeout"`
			ConnectionSelector string `yaml:"connectionSelector"`
		}
		Sqlite struct {
			Name string `yaml:"name"`
		}
	} `yaml:"store"`

	Portable struct {
		PythonBin string `yaml:"pythonBin"`
	} `yaml:"portable"`
}

func InitConsulConf(confFile *string) error {
	if nil == confFile {
		panic(errors.New("config file is null"))
	}

	b, err := ioutil.ReadFile(*confFile)
	if err != nil {
		panic(err)
	}

	kc := KuiperConf{
		Rule: api.RuleOption{
			LateTol:            1000,
			Concurrency:        1,
			BufferLength:       1024,
			CheckpointInterval: 300000, //5 minutes
			SendError:          true,
		},
	}

	if e := yaml.Unmarshal(b, &kc); e != nil {
		return e
	}

	Config = &kc

	return nil
}

func InitKuiperPropertyConf(confFile *string, kuiperPropertyData []byte) error {
	if 0 == len(kuiperPropertyData) && nil != confFile {
		b, err := ioutil.ReadFile(*confFile)
		if err != nil {
			panic(err)
		}
		kuiperPropertyData = b
	}

	if e := yaml.Unmarshal(kuiperPropertyData, Config); e != nil {
		return e
	}

	if 0 == len(Config.Basic.Ip) {
		Config.Basic.Ip = "0.0.0.0"
	}
	if 0 == len(Config.Basic.RestIp) {
		Config.Basic.RestIp = "0.0.0.0"
	}

	if Config.Basic.Debug {
		Log.SetLevel(logrus.DebugLevel)
	}

	if Config.Basic.FileLog {
		file := Config.Log.FilesPath + "/" + logFileName
		logWriter, err := rotatelogs.New(
			file+".%Y-%m-%d_%H-%M-%S",
			rotatelogs.WithLinkName(file),
			rotatelogs.WithRotationTime(time.Hour*time.Duration(Config.Basic.RotateTime)),
			rotatelogs.WithMaxAge(time.Hour*time.Duration(Config.Basic.MaxAge)),
		)

		if err != nil {
			fmt.Println("Failed to init log file settings..." + err.Error())
			Log.Infof("Failed to log to file, using default stderr.")
		} else if Config.Basic.ConsoleLog {
			mw := io.MultiWriter(os.Stdout, logWriter)
			Log.SetOutput(mw)
		} else if !Config.Basic.ConsoleLog {
			Log.SetOutput(logWriter)
		}
	} else if Config.Basic.ConsoleLog {
		Log.SetOutput(os.Stdout)
	}

	if Config.Store.Type == "redis" && Config.Store.Redis.ConnectionSelector != "" {
		if err := RedisStorageConSelectorApply(Config.Store.Redis.ConnectionSelector, Config); err != nil {
			Log.Fatal(err)
		}
	}

	if Config.Portable.PythonBin == "" {
		Config.Portable.PythonBin = "python"
	}

	return nil
}

func init() {
	InitLogger()
	InitClock()
}
