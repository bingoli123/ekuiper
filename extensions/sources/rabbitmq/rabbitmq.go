package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/lf-edge/ekuiper/pkg/api"
	"github.com/lf-edge/ekuiper/pkg/message"
	"github.com/streadway/amqp"
	"runtime"
	"strings"
	"time"
)

type rabbitmqSource struct {
	conn           *amqp.Connection
	channel        *amqp.Channel
	queue          amqp.Queue
	deliverMsgList <-chan amqp.Delivery

	server        string
	username      string
	password      string
	vhost         string
	bindKeys      string
	exchange      string
	queueName     string
	messageFormat string
	cancel        context.CancelFunc
}

func (s *rabbitmqSource) Configure(topic string, props map[string]interface{}) error {
	{
		srv, ok := props["server"]
		if !ok {
			return fmt.Errorf("rabbitmq source is missing property server")
		}
		s.server = srv.(string)
	}

	{
		username, ok := props["username"]
		if !ok {
			return fmt.Errorf("rabbitmq source is missing property username")
		}
		s.username = username.(string)
	}

	{
		password, ok := props["password"]
		if !ok {
			return fmt.Errorf("rabbitmq source is missing property password")
		}
		s.password = password.(string)
	}

	{
		vhost, ok := props["vhost"]
		if !ok {
			return fmt.Errorf("rabbitmq source is missing property vhost")
		}
		s.vhost = vhost.(string)
	}

	{
		bindKeys, ok := props["bind_keys"]
		if !ok {
			return fmt.Errorf("rabbitmq source is missing property bind keys")
		}
		s.bindKeys = bindKeys.(string)
	}

	{
		exchange, ok := props["exchange"]
		if !ok {
			return fmt.Errorf("rabbitmq source is missing property exchange")
		}
		s.exchange = exchange.(string)
	}

	{
		s.queueName = uuid.NewString()
	}

	{
		f, ok := props["format"]
		if !ok {
			s.messageFormat = message.FormatJson
		} else {
			s.messageFormat = f.(string)
		}
	}

	return nil
}

func (s *rabbitmqSource) Open(ctx api.StreamContext, consumer chan<- api.SourceTuple, errCh chan<- error) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(identifyPanic())
		}
	}()

	logger := ctx.GetLogger()
	logger.Infof("source server=%s, rabbitmq source come in, config=%+v", s.server, s)

	var err error
	url := fmt.Sprintf("amqp://%s:%s@%s%s", s.username, s.password, s.server, s.vhost)
	s.conn, err = amqp.Dial(url)
	if err != nil {
		errCh <- fmt.Errorf("rabbitmq source fails to connect to %s: %v", s.server, err)
	}

	s.channel, err = s.conn.Channel()
	if err != nil {
		errCh <- fmt.Errorf("rabbitmq source fails to create channel %s: %v", s.server, err)
	}

	s.queue, err = s.channel.QueueDeclare(
		s.queueName, // name
		false,       // durable
		true,        // delete when usused
		true,        // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	if err != nil {
		errCh <- fmt.Errorf("rabbitmq source fails to declare queue %s: %v", s.server, err)
	}

	logger.Infof("rabbitmq source[%s] create queue [%s] success", s.queueName)

	var bindKeys []string
	bindKeys = strings.Split(s.bindKeys, ",")
	for _, key := range bindKeys {
		err = s.channel.QueueBind(
			s.queueName, // queue name
			key,         // routing key
			s.exchange,  // exchange
			false,
			nil)
		if err != nil {
			errCh <- fmt.Errorf("rabbitmq source fails to bind key[%s] server=%s: %v", key, s.server, err)
		}
	}

	s.deliverMsgList, err = s.channel.Consume(
		s.queueName, // queue
		"",          // consumer
		true,        // auto ack
		false,       // exclusive
		false,       // no local
		false,       // no wait
		nil,         // args
	)
	if err != nil {
		errCh <- fmt.Errorf("rabbitmq source fails to create consume %s: %v", s.server, err)
	}

	exeCtx, cancel := ctx.WithCancel()
	s.cancel = cancel
	logger.Infof("rabbitmq source[%s],queue[%s] start to listen msg", s.server, s.queueName)
	for {
		select {
		case msg := <-s.deliverMsgList:
			if len(msg.Body) > 0 {
				meta := make(map[string]interface{})
				result, e := message.Decode(msg.Body, s.messageFormat)
				if e != nil {
					logger.Errorf("Invalid data format, cannot decode %v to %s format with error %s", msg, s.messageFormat, e)
					return
				} else {
					consumer <- api.NewDefaultSourceTuple(result, meta)
				}

				logger.Debugf("server=%s, queue=%s, rabbitmq source receive msg=%+v", s.server, s.queueName, msg)
			} else {
				// 这里之所以这么处理，是因为当mq连接上之后，假如中间断开，这个队列会死循环往外吐空消息，导致cpu占用过高，因此没有消息的时候
				// 可以睡一秒等待下
				time.Sleep(time.Second)
			}
		case <-exeCtx.Done():
			{
				logger.Infof("rabbitmq server=%s, source done,queue=%s", s.server, s.queueName)

				return
			}
		}
	}
}

func (s *rabbitmqSource) Close(ctx api.StreamContext) error {
	logger := ctx.GetLogger()

	if s.cancel != nil {
		s.cancel()
	}

	if nil != s.channel {
		if _, err := s.channel.QueueDelete(s.queueName, false, false, true); err != nil {
			logger.Errorf("source server=%s delete queue %s error=%s", s.server, s.queueName, err.Error())
		} else {
			logger.Infof("source server=%s delete queue %s success", s.server, s.queueName)
		}

		_ = s.channel.Close()
	}

	if nil != s.conn {
		_ = s.conn.Close()
	}

	logger.Infof("rabbitmq source close success")

	return nil
}

func Rabbitmq() api.Source {
	return &rabbitmqSource{}
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
