package kafka

import (
	"context"
	"encoding/json"
	"time"

	ckafka "github.com/confluentinc/confluent-kafka-go/kafka"
)

type Producer struct {
	p *ckafka.Producer
}

func NewProducer(brokers string) (*Producer, error) {
	p, err := ckafka.NewProducer(&ckafka.ConfigMap{
		"bootstrap.servers": brokers,
		"enable.idempotence": true,
		"acks":               "all",
		"retries":            10,
		"delivery.timeout.ms": 120000,
	})
	if err != nil {
		return nil, err
	}
	return &Producer{p: p}, nil
}

func (pr *Producer) Close() {
	pr.p.Flush(5000)
	pr.p.Close()
}

func (pr *Producer) PublishJSON(ctx context.Context, topic string, key string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	delivery := make(chan ckafka.Event, 1)
	defer close(delivery)

	err = pr.p.Produce(&ckafka.Message{
		TopicPartition: ckafka.TopicPartition{Topic: &topic, Partition: int32(ckafka.PartitionAny)},
		Key:            []byte(key),
		Value:          b,
		Timestamp:      time.Now(),
	}, delivery)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case ev := <-delivery:
		m := ev.(*ckafka.Message)
		return m.TopicPartition.Error
	}
}

