// Copyright 2022 EMQ Technologies Co., Ltd.
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

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/lf-edge/ekuiper/internal/binder/function"
	"github.com/lf-edge/ekuiper/internal/binder/io"
	"github.com/lf-edge/ekuiper/internal/binder/meta"
	"github.com/lf-edge/ekuiper/internal/conf"
	"github.com/lf-edge/ekuiper/internal/pkg/store"
	"github.com/lf-edge/ekuiper/internal/processor"
	"github.com/lf-edge/ekuiper/internal/server/consul"
	"github.com/lf-edge/ekuiper/internal/topo/connection/factory"
	"gopkg.in/yaml.v3"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

var (
	logger          = conf.Log
	startTimeStamp  int64
	version         = ""
	ruleProcessor   *processor.RuleProcessor
	streamProcessor *processor.StreamProcessor
)

func GetCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return ""
	}
	return strings.Replace(dir, "\\", "/", -1)
}
func StartUp(ConfFile *string, Version, LoadFileType string) {
	version = Version
	conf.LoadFileType = LoadFileType
	startTimeStamp = time.Now().Unix()
	if err := conf.InitConsulConf(ConfFile); err != nil {
		panic(err)
	}

	if 0 == len(conf.Config.Service.DataRoot) {
		conf.Config.Service.DataRoot = GetCurrentDirectory() + "/../"
	}

	if conf.Config.ConsulConnection.Enable {
		// get property config
		if true {
			var configData []byte

			for 0 == len(configData) {
				var err error
				if configData, err = consul.GetValue(conf.Config.ConsulConnection.Host,
					conf.Config.ConsulConnection.Port,
					conf.Config.ConsulConnection.Datacenter,
					conf.Config.ConsulConnection.PropertyKvPath); err != nil {
					fmt.Printf("get property config from consul[%+v] failed, try again... \n", conf.Config.ConsulConnection)
					time.Sleep(time.Second * 3)
				}
			}

			fmt.Println("get property from consul success")
			if err := conf.InitKuiperPropertyConf(nil, configData); err != nil {
				panic(err)
			}
		}

		// get kong config
		if true {
			var configData []byte

			for 0 == len(configData) {
				var err error
				if configData, err = consul.GetValue(conf.Config.ConsulConnection.Host,
					conf.Config.ConsulConnection.Port,
					conf.Config.ConsulConnection.Datacenter,
					conf.Config.Kong.Remote); err != nil {
					fmt.Printf("get kong config from consul[%+v] failed, try again... \n", conf.Config.ConsulConnection)
					time.Sleep(time.Second * 3)
				}
			}

			if err := json.Unmarshal(configData, &conf.Config.Kong.Local); err != nil {
				panic(err)
			}

			fmt.Println("get kong config from consul success")
		}

		var domainData []byte
		if false {
			for 0 == len(domainData) {
				var err error
				if domainData, err = consul.GetValue(conf.Config.ConsulConnection.Host,
					conf.Config.ConsulConnection.Port,
					conf.Config.ConsulConnection.Datacenter,
					conf.Config.Domains.Remote); err != nil {
					fmt.Println("get domain config from consul failed, try again...")
					time.Sleep(time.Second * 3)
				}
			}
		}

		var domainInfo []conf.DomainInfo
		if false {
			if err := yaml.Unmarshal(domainData, &domainInfo); err != nil {
				panic(err)
			}
		}

		if true {
			// register consul service
			c := consul.NewConsul()

			var ServiceDomain string
			if len(domainInfo) > 0 {
				ServiceDomain = domainInfo[0].DomainUid
			}

			if err := c.Init(
				conf.Config.ConsulConnection.Host,
				conf.Config.ConsulConnection.Port,
				conf.Config.ConsulConnection.Datacenter,
				30,
				60,
				conf.Config.Service.NodeHost,
				conf.Config.Basic.RestPort,
				conf.Config.ServiceUid,
				"kuiperd",
				ServiceDomain); err != nil {
				panic(err)
			}

			if err := c.RegisterService(); err != nil {
				panic(err)
			}

			c.HeartCheck()
		}

		if true {
			// reload config data
			go func() {
				var property []byte

				for {
					if propertyDataTmp, err := consul.GetValue(conf.Config.ConsulConnection.Host,
						conf.Config.ConsulConnection.Port,
						conf.Config.ConsulConnection.Datacenter,
						conf.Config.ConsulConnection.PropertyKvPath); err != nil {
						fmt.Println("reload get property config from consul failed, try again...")
						time.Sleep(time.Second * 5)
						continue
					} else {
						if 0 == len(property) {
							property = propertyDataTmp
						}

						if !bytes.Equal(property, propertyDataTmp) {
							os.Exit(-1)
						}
					}

					time.Sleep(time.Second * 10)
				}
			}()
		}
	} else {
		if err := conf.InitKuiperPropertyConf(ConfFile, nil); err != nil {
			panic(err)
		}
	}

	if err := KongInit(conf.Config.Kong.Local.Manage.Host,
		conf.Config.Kong.Local.Manage.Port,
		conf.Config.Basic.RestPort,
		conf.Config.Kong.Local.KongPlugins); err != nil {
		panic(err)
	}

	factory.InitClientsFactory()

	err := store.SetupWithKuiperConfig(conf.Config)
	if err != nil {
		panic(err)
	}

	ruleProcessor = processor.NewRuleProcessor()
	streamProcessor = processor.NewStreamProcessor()

	// register all extensions
	for k, v := range components {
		logger.Infof("register component %s", k)
		v.register()
	}

	// Bind the source, function, sink
	sort.Sort(entries)
	err = function.Initialize(entries)
	if err != nil {
		panic(err)
	}
	err = io.Initialize(entries)
	if err != nil {
		panic(err)
	}
	meta.Bind()

	registry = &RuleRegistry{internal: make(map[string]*RuleState)}
	//Start rules
	if rules, err := ruleProcessor.GetAllRules(); err != nil {
		logger.Infof("Start rules error: %s", err)
	} else {
		logger.Info("Starting rules")
		var reply string
		for _, rule := range rules {
			//err = server.StartRule(rule, &reply)
			reply = recoverRule(rule)
			if 0 != len(reply) {
				logger.Info(reply)
			}
		}
	}

	//Start rest service
	srvRest := createRestServer(conf.Config.Basic.RestIp, conf.Config.Basic.RestPort, conf.Config.Basic.Authentication)
	go func() {
		var err error
		if conf.Config.Basic.RestTls == nil {
			err = srvRest.ListenAndServe()
		} else {
			err = srvRest.ListenAndServeTLS(conf.Config.Basic.RestTls.Certfile, conf.Config.Basic.RestTls.Keyfile)
		}
		if err != nil && err != http.ErrServerClosed {
			logger.Errorf("Error serving rest service: %s", err)
		}
	}()

	// Start extend services
	for k, v := range servers {
		logger.Infof("start service %s", k)
		v.serve()
	}

	//Startup message
	restHttpType := "http"
	if conf.Config.Basic.RestTls != nil {
		restHttpType = "https"
	}
	msg := fmt.Sprintf("Serving kuiper (version - %s) on port %d, and restful api on %s://%s:%d. \n", Version, conf.Config.Basic.Port, restHttpType, conf.Config.Basic.RestIp, conf.Config.Basic.RestPort)
	logger.Info(msg)
	fmt.Printf(msg)

	//Stop the services
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
	<-sigint

	if err = srvRest.Shutdown(context.TODO()); err != nil {
		logger.Errorf("rest server shutdown error: %v", err)
	}
	logger.Info("rest server successfully shutdown.")

	// close extend services
	for k, v := range servers {
		logger.Infof("close service %s", k)
		v.close()
	}

	os.Exit(0)
}
