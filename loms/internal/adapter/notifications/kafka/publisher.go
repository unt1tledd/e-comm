package kafka

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/port"
	"github.com/segmentio/kafka-go"
)

type Publisher struct {
	writer *kafka.Writer
}

func NewPublisher(brokers []string, topic string) *Publisher {
	return &Publisher{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Topic:                  topic,
			Balancer:               &kafka.Hash{},
			AllowAutoTopicCreation: true,
		},
	}
}

func (p *Publisher) Close() error {
	return p.writer.Close()
}

func (p *Publisher) SendOrderStatusChangedNotification(
	ctx context.Context,
	userID,
	orderID int64,
	status port.OrderStatus,
) error {
	body := port.OrderStatusChangedNotification{
		OrderID: orderID,
		UserID:  userID,
		Status:  status,
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(strconv.FormatInt(orderID, 10)),
		Value: raw,
	})
}
