package main

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/lf-edge/ekuiper/pkg/api"
	"github.com/lf-edge/ekuiper/pkg/errorx"

	"github.com/streadway/amqp"
	"runtime"
	"strings"
)

type rabbitmqSink struct {
	conn    *amqp.Connection
	channel *amqp.Channel

	server       string
	username     string
	password     string
	vhost        string
	exchange     string
	exchangeType string
	routeKeys    string

	uid string
}

func (s *rabbitmqSink) Configure(props map[string]interface{}) error {
	{
		server, ok := props["server"]
		if !ok {
			return fmt.Errorf("rabbitmq sink is missing property server")
		}
		s.server, ok = server.(string)
		if !ok {
			return fmt.Errorf("rabbitmq sink property server %v is not a string", server)
		}
	}

	{
		username, ok := props["username"]
		if !ok {
			return fmt.Errorf("rabbitmq sink is missing property username")
		}
		s.username = username.(string)
	}

	{
		password, ok := props["password"]
		if !ok {
			return fmt.Errorf("rabbitmq sink is missing property password")
		}
		s.password = password.(string)
	}

	{
		vhost, ok := props["vhost"]
		if !ok {
			return fmt.Errorf("rabbitmq sink is missing property vhost")
		}
		s.vhost = vhost.(string)
	}

	{
		routeKeys, ok := props["route_keys"]
		if !ok {
			return fmt.Errorf("rabbitmq sink is missing property route keys")
		}
		s.routeKeys = routeKeys.(string)
	}

	{
		exchange, ok := props["exchange"]
		if !ok {
			return fmt.Errorf("rabbitmq sink is missing property exchange")
		}
		s.exchange = exchange.(string)
	}

	{
		exchangeType, ok := props["exchange_type"]
		if !ok {
			return fmt.Errorf("rabbitmq sink is missing property exchange type")
		}
		s.exchangeType = exchangeType.(string)
	}

	return nil
}

func (s *rabbitmqSink) Open(ctx api.StreamContext) (err error) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(identifyPanic())
		}
	}()

	s.uid = uuid.NewString()

	logger := ctx.GetLogger()
	logger.Infof("sink uid=%s, rabbitmq sink come in, config=%+v", s.uid, s)

	url := fmt.Sprintf("amqp://%s:%s@%s%s", s.username, s.password, s.server, s.vhost)
	s.conn, err = amqp.Dial(url)
	if err != nil {
		return fmt.Errorf("rabbitmq sink fails to connect to %s: %v", s.server, err)
	}

	s.channel, err = s.conn.Channel()
	if err != nil {
		return fmt.Errorf("rabbitmq sink fails to create channel %s: %v", s.server, err)
	}

	if err = s.channel.ExchangeDeclare(
		s.exchange,     // name
		s.exchangeType, // type
		true,           // durable
		false,          // auto-deleted
		false,          // internal
		false,          // no-wait
		nil,            // arguments
	); err != nil {
		return err
	}

	logger.Debugf("rabbitmq sink open")

	return nil
}

func (s *rabbitmqSink) Collect(ctx api.StreamContext, item interface{}) (err error) {
	logger := ctx.GetLogger()
	if msg, _, err := ctx.TransformOutput(item); err == nil {
		logger.Debugf("sink uid=%s, rabbitmq sink receive %+v", s.uid, item)

		var routeKeys []string
		routeKeys = strings.Split(s.routeKeys, ",")
		for _, v := range routeKeys {
			if err = s.channel.Publish(
				s.exchange, // exchange
				v,          // routing key
				false,      // mandatory
				false,      // immediate
				amqp.Publishing{
					Body: msg,
				}); err != nil {
				return err
			}
		}
	} else {
		logger.Debug("rabbitmq sink receive non byte data %v", item)
	}

	if err != nil {
		logger.Errorf("send to rabbitmq error %v", err)
		return fmt.Errorf("%s:%s", errorx.IOErr, err.Error())
	}
	return
}

func (s *rabbitmqSink) Close(ctx api.StreamContext) error {
	if nil != s.channel {
		_ = s.channel.Close()
	}

	if nil != s.conn {
		_ = s.conn.Close()
	}

	logger := ctx.GetLogger()
	logger.Infof("rabbitmq sink close success")

	return nil
}

func Rabbitmq() api.Sink {
	return &rabbitmqSink{}
}

func identifyPanic() string {
	var name, file string
	var line int
	var pc [16]uintptr

	n := runtime.Callers(3, pc[:])
	for _, pc := range pc[:n] {
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		file, line = fn.FileLine(pc)
		name = fn.Name()
		if !strings.HasPrefix(name, "runtime.") {
			break
		}
	}

	switch {
	case name != "":
		return fmt.Sprintf("%v:%v", name, line)
	case file != "":
		return fmt.Sprintf("%v:%v", file, line)
	}

	return fmt.Sprintf("pc:%x", pc)
}
