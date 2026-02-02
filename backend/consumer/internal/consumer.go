package kafka

import (
	"time"

	ckafka "github.com/confluentinc/confluent-kafka-go/kafka"
)

type Consumer struct {
	c *ckafka.Consumer
}

func NewConsumer(brokers, groupID string) (*Consumer, error) {
	c, err := ckafka.NewConsumer(&ckafka.ConfigMap{
		"bootstrap.servers": brokers,
		"group.id":          groupID,
		"auto.offset.reset": "earliest",
		"enable.auto.commit": true,
	})
	if err != nil {
		return nil, err
	}
	return &Consumer{c: c}, nil
}

func (co *Consumer) Close() error {
	return co.c.Close()
}

func (co *Consumer) Subscribe(topic string) error {
	return co.c.SubscribeTopics([]string{topic}, nil)
}

func (co *Consumer) Poll(ms int) (any, error) {
	return co.c.ReadMessage(time.Duration(ms) * time.Millisecond)
}
