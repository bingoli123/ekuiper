package consul

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"strconv"
	"time"
)

type Consul struct {
	host                           string
	port                           int
	datacenter                     string
	ttl                            int
	deregisterCriticalServiceAfter int
	gatewayServiceAddr             string
	gatewayServicePort             int
	serviceUID                     string
	serviceName                    string
	serviceDomain                  string
	checkID                        string
	registerServiceID              string
	client                         *api.Client
	bRunning                       bool
}

func NewConsul() *Consul {
	return &Consul{
		bRunning: true,
	}
}

func (c *Consul) Init(Host string,
	Port int,
	Datacenter string,
	TTL int,
	DeregisterCriticalServiceAfter int,
	GatewayServiceAddr string,
	GatewayServicePort int,
	ServiceUID string,
	ServiceName string,
	ServiceDomain string) error {

	if true {
		c.host = Host
		c.port = Port
		c.datacenter = Datacenter
		c.ttl = TTL
		c.deregisterCriticalServiceAfter = DeregisterCriticalServiceAfter
		c.gatewayServiceAddr = GatewayServiceAddr
		c.gatewayServicePort = GatewayServicePort
		c.serviceUID = ServiceUID
		c.serviceName = ServiceName
		c.serviceDomain = ServiceDomain
	}

	if true {
		config := api.DefaultConfig()
		config.Address = fmt.Sprintf("%s:%d", c.host, c.port)
		config.Datacenter = c.datacenter

		client, err := api.NewClient(config)
		if err != nil {
			return err
		}
		c.client = client
	}

	return nil
}

func (c *Consul) RegisterService() error {
	if true {
		c.checkID = "check:" + c.serviceDomain + "-" + c.serviceUID
		c.registerServiceID = c.serviceDomain + "-" + c.serviceUID
		reg := &api.AgentServiceRegistration{
			Name: c.serviceName,
			ID:   c.registerServiceID,
			Port: c.port,
			Check: &api.AgentServiceCheck{
				CheckID:                        c.checkID,
				TTL:                            strconv.Itoa(c.ttl) + "s",
				DeregisterCriticalServiceAfter: strconv.Itoa(c.deregisterCriticalServiceAfter) + "s",
			},
		}

		reg.Address = fmt.Sprintf("%s:%d", c.host, c.port)

		err := c.client.Agent().ServiceRegister(reg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Consul) HeartCheck() {
	go func() {
		for c.bRunning {
			if err := c.client.Agent().PassTTL(c.checkID, "passing-"+time.Now().String()); err != nil {
				_ = c.RegisterService()
			}

			time.Sleep(time.Second * 10)
		}

	}()
}

func (c *Consul) UnInit() error {
	c.bRunning = false

	if err := c.client.Agent().ServiceDeregister(c.registerServiceID); err != nil {
		return err
	}

	return nil
}

func GetValue(host string, port int, datacenter string, key string) ([]byte, error) {
	config := api.DefaultConfig()
	config.Address = fmt.Sprintf("%s:%d", host, port)
	config.Datacenter = datacenter

	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	pair, _, err := client.KV().Get(key, nil)
	if err != nil {
		return nil, err
	}

	if pair == nil {
		return nil, nil
	}

	return pair.Value, nil
}
