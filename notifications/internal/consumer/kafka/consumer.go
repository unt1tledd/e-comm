package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	ckafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/config"
	"go.uber.org/zap"
)

//go:generate mockgen -source=consumer.go -destination=mocks/consumer_mocks.go -package=mocks

type (
	notifier interface {
		NotifyOrderStatusChanged(
			ctx context.Context,
			userID,
			orderID int64,
			status string,
		) error
	}
)

type eventPayload struct {
	UserID  int64  `json:"user_id"`
	OrderID int64  `json:"order_id"`
	Status  string `json:"status"`
}

var errDLQProducerDisabled = errors.New("kafka dlq producer disabled")

func RunConsumer(
	ctx context.Context,
	logger *zap.Logger,
	cfg *config.Config,
	notifier notifier,
) error {
	brokers := cfg.KafkaBrokerAddrs()
	if len(brokers) == 0 {
		return errors.New("empty kafka brokers")
	}

	bootstrapServersKey := "bootstrap.servers"
	consumer, err := ckafka.NewConsumer(&ckafka.ConfigMap{
		bootstrapServersKey:        joinBrokers(brokers),
		"group.id":                 cfg.Kafka.ConsumerGroup,
		"auto.offset.reset":        "earliest",
		"enable.auto.commit":       false,
		"enable.auto.offset.store": false,
		"session.timeout.ms":       cfg.Kafka.SessionTimeoutMs,
	})

	if err != nil {
		return err
	}

	defer func() {
		if err := consumer.Close(); err != nil {
			logger.Error("close kafka consumer", zap.Error(err))
		}
	}()

	var producer *ckafka.Producer
	defer func() {
		if producer != nil {
			producer.Close()
		}
	}()

	if err := consumer.SubscribeTopics([]string{cfg.Kafka.Topic}, nil); err != nil {
		return err
	}

	logger.Info("kafka consumer started",
		zap.Strings("brokers", brokers),
		zap.String("topic", cfg.Kafka.Topic),
		zap.String("dlq_topic", cfg.Kafka.DLQTopic),
		zap.String("group", cfg.Kafka.ConsumerGroup),
		zap.Int("batch_size", batchSize(cfg)),
	)

	for {
		select {
		case <-ctx.Done():
			logger.Info("kafka consumer stopped", zap.Error(ctx.Err()))
			return nil
		default:
		}

		batch, ok := pollBatch(ctx, logger, cfg, consumer)
		if !ok {
			return nil
		}

		if len(batch) == 0 {
			continue
		}

		if !processBatchWithoutCommit(ctx, logger, cfg, &producer, brokers, bootstrapServersKey, notifier, batch) {
			if ctx.Err() != nil {
				logger.Info("kafka consumer stopped", zap.Error(ctx.Err()))
				return nil
			}
			time.Sleep(cfg.Kafka.RetryBackoff)
			continue
		}

		if _, err := consumer.CommitOffsets(commitOffsets(batch)); err != nil {
			logger.Error("kafka batch commit",
				zap.Error(err),
				zap.Int("batch_size", len(batch)),
			)

			time.Sleep(cfg.Kafka.RetryBackoff)
			continue
		}

		logger.Info("kafka batch committed", zap.Int("batch_size", len(batch)))
	}
}

type batchConsumer interface {
	Poll(timeoutMs int) ckafka.Event
	Assign(partitions []ckafka.TopicPartition) error
	Unassign() error
}

type dlqWriter interface {
	Produce(msg *ckafka.Message, deliveryChan chan ckafka.Event) error
}

func pollBatch(
	ctx context.Context,
	logger *zap.Logger,
	cfg *config.Config,
	consumer batchConsumer,
) ([]*ckafka.Message, bool) {
	batch := make([]*ckafka.Message, 0, batchSize(cfg))

	for len(batch) < batchSize(cfg) {
		select {
		case <-ctx.Done():
			logger.Info("kafka consumer stopped", zap.Error(ctx.Err()))
			return batch, false
		default:
		}

		pollTimeoutMs := cfg.Kafka.PollTimeoutMs
		if len(batch) > 0 {
			pollTimeoutMs = 0
		}

		ev := consumer.Poll(pollTimeoutMs)
		if ev == nil {
			return batch, true
		}

		switch e := ev.(type) {
		case *ckafka.Message:
			logger.Info("message fetched",
				zap.String("topic", topicName(e)),
				zap.Int32("partition", e.TopicPartition.Partition),
				zap.Int64("offset", int64(e.TopicPartition.Offset)),
				zap.ByteString("key", e.Key),
				zap.ByteString("value", e.Value),
			)

			batch = append(batch, e)

		case ckafka.Error:
			logger.Warn("kafka poll error",
				zap.String("code", e.Code().String()),
				zap.Error(e),
			)

			time.Sleep(cfg.Kafka.RetryBackoff)

		case ckafka.AssignedPartitions:
			logger.Info("kafka partitions assigned",
				zap.Any("partitions", e.Partitions),
			)

			if err := consumer.Assign(e.Partitions); err != nil {
				logger.Error("kafka assign partitions", zap.Error(err))
			}

		case ckafka.RevokedPartitions:
			logger.Info("kafka partitions revoked",
				zap.Any("partitions", e.Partitions),
			)

			if err := consumer.Unassign(); err != nil {
				logger.Error("kafka unassign partitions", zap.Error(err))
			}

		default:
		}
	}

	return batch, true
}

func processBatchWithoutCommit(
	ctx context.Context,
	logger *zap.Logger,
	cfg *config.Config,
	producer **ckafka.Producer,
	brokers []string,
	bootstrapServersKey string,
	notifier notifier,
	batch []*ckafka.Message,
) bool {
	for _, message := range batch {
		ok, _ := processMessageWithoutCommitWithDLQProducer(ctx, logger, cfg, producer, brokers, bootstrapServersKey, notifier, message)
		if !ok {
			return false
		}
	}

	return true
}

func processMessageWithoutCommitWithDLQProducer(
	ctx context.Context,
	logger *zap.Logger,
	cfg *config.Config,
	producer **ckafka.Producer,
	brokers []string,
	bootstrapServersKey string,
	notifier notifier,
	message *ckafka.Message,
) (bool, bool) {
	var payload eventPayload
	if unmarshalErr := json.Unmarshal(message.Value, &payload); unmarshalErr != nil {
		logger.Error("kafka message unmarshal",
			zap.Error(unmarshalErr),
			zap.ByteString("raw", message.Value),
		)

		producerClient, err := getDLQProducer(producer, brokers, bootstrapServersKey)
		if err != nil {
			if errors.Is(err, errDLQProducerDisabled) {
				return true, false
			}

			logger.Error("create kafka dlq producer", zap.Error(err))
			return false, false
		}

		if producerClient == nil {
			return true, false
		}

		if produceErr := sendToDLQ(ctx, cfg, producerClient, message, unmarshalErr); produceErr != nil {
			logger.Error("send kafka message to dlq",
				zap.Error(produceErr),
				zap.String("topic", topicName(message)),
				zap.Int32("partition", message.TopicPartition.Partition),
				zap.Int64("offset", int64(message.TopicPartition.Offset)),
			)

			return false, false
		}

		logger.Info("message sent to dlq",
			zap.String("topic", topicName(message)),
			zap.Int32("partition", message.TopicPartition.Partition),
			zap.Int64("offset", int64(message.TopicPartition.Offset)),
		)

		return true, false
	}

	return notifyWithRetry(ctx, logger, cfg, notifier, payload), true
}

func notifyWithRetry(
	ctx context.Context,
	logger *zap.Logger,
	cfg *config.Config,
	notifier notifier,
	payload eventPayload,
) bool {
	for {
		if err := notifier.NotifyOrderStatusChanged(ctx, payload.UserID, payload.OrderID, payload.Status); err != nil {
			logger.Error("notify order status",
				zap.Error(err),
				zap.Int64("user_id", payload.UserID),
				zap.Int64("order_id", payload.OrderID),
				zap.String("status", payload.Status),
			)

			if ctx.Err() != nil {
				return false
			}

			select {
			case <-ctx.Done():
				return false
			case <-time.After(cfg.Kafka.RetryBackoff):
				continue
			}
		}

		return true
	}
}

func getDLQProducer(producer **ckafka.Producer, brokers []string, bootstrapServersKey string) (dlqWriter, error) {
	if producer == nil {
		return nil, errDLQProducerDisabled
	}

	if *producer != nil {
		return *producer, nil
	}

	created, err := ckafka.NewProducer(&ckafka.ConfigMap{
		bootstrapServersKey: joinBrokers(brokers),
	})
	if err != nil {
		return nil, err
	}

	*producer = created
	return created, nil
}

func sendToDLQ(
	ctx context.Context,
	cfg *config.Config,
	producer dlqWriter,
	message *ckafka.Message,
	cause error,
) error {
	if cfg.Kafka.DLQTopic == "" {
		return errors.New("empty kafka dlq topic")
	}

	deliveryCh := make(chan ckafka.Event, 1)

	err := producer.Produce(&ckafka.Message{
		TopicPartition: ckafka.TopicPartition{
			Topic:     &cfg.Kafka.DLQTopic,
			Partition: ckafka.PartitionAny,
		},
		Key:   message.Key,
		Value: message.Value,
		Headers: append(message.Headers,
			ckafka.Header{Key: "x-original-topic", Value: []byte(topicName(message))},
			ckafka.Header{Key: "x-original-partition", Value: []byte(strconv.Itoa(int(message.TopicPartition.Partition)))},
			ckafka.Header{Key: "x-original-offset", Value: []byte(strconv.FormatInt(int64(message.TopicPartition.Offset), 10))},
			ckafka.Header{Key: "x-error", Value: []byte(cause.Error())},
		),
	}, deliveryCh)
	if err != nil {
		return err
	}

	timer := time.NewTimer(cfg.Kafka.DLQTimeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return errors.New("kafka dlq produce timeout")
	case ev := <-deliveryCh:
		delivery, ok := ev.(*ckafka.Message)
		if !ok {
			return fmt.Errorf("unexpected kafka dlq delivery event: %T", ev)
		}

		return delivery.TopicPartition.Error
	}
}

func commitOffsets(batch []*ckafka.Message) []ckafka.TopicPartition {
	offsets := make(map[string]ckafka.TopicPartition)

	for _, message := range batch {
		topic := topicName(message)
		key := fmt.Sprintf("%s:%d", topic, message.TopicPartition.Partition)
		offset := message.TopicPartition
		offset.Offset++
		offset.Error = nil

		current, ok := offsets[key]
		if !ok || offset.Offset > current.Offset {
			offsets[key] = offset
		}
	}

	result := make([]ckafka.TopicPartition, 0, len(offsets))
	for _, offset := range offsets {
		result = append(result, offset)
	}

	return result
}

func batchSize(cfg *config.Config) int {
	if cfg.Kafka.BatchSize <= 0 {
		return 1
	}

	return cfg.Kafka.BatchSize
}

func topicName(message *ckafka.Message) string {
	if message.TopicPartition.Topic == nil {
		return ""
	}

	return *message.TopicPartition.Topic
}

func joinBrokers(brokers []string) string {
	return strings.Join(brokers, ",")
}
