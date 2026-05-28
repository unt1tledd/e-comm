package kafka

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	ckafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/config"
	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/consumer/kafka/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestNotifyWithRetry(t *testing.T) {
	t.Parallel()

	errAny := errors.New("any error")

	tests := []struct {
		name      string
		errs      []error
		cancelCtx bool
		wantOK    bool
		wantCalls int
	}{
		{
			name:      "success",
			wantOK:    true,
			wantCalls: 1,
		},
		{
			name:      "retry then success",
			errs:      []error{errAny, nil},
			wantOK:    true,
			wantCalls: 2,
		},
		{
			name:      "context cancelled",
			errs:      []error{errAny},
			cancelCtx: true,
			wantOK:    false,
			wantCalls: 1,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			notifier := mocks.NewMocknotifier(ctrl)
			if tt.cancelCtx {
				cancel()
			}

			calls := make([]any, 0, tt.wantCalls)
			for i := 0; i < tt.wantCalls; i++ {
				var err error
				if i < len(tt.errs) {
					err = tt.errs[i]
				}

				calls = append(calls, notifier.EXPECT().
					NotifyOrderStatusChanged(gomock.Any(), int64(900001), int64(42), "awaiting_payment").
					Return(err))
			}
			gomock.InOrder(calls...)

			cfg := &config.Config{}
			cfg.Kafka.RetryBackoff = time.Nanosecond

			ok := notifyWithRetry(ctx, zap.NewNop(), cfg, notifier, eventPayload{
				UserID:  900001,
				OrderID: 42,
				Status:  "awaiting_payment",
			})

			require.Equal(t, tt.wantOK, ok)
		})
	}
}

func TestProcessMessageWithoutCommit(t *testing.T) {
	t.Parallel()

	topic := "order_status_notifications"
	message := &ckafka.Message{
		TopicPartition: ckafka.TopicPartition{
			Topic:     &topic,
			Partition: 1,
			Offset:    10,
		},
		Value: []byte(`{"user_id":900001,"order_id":42,"status":"paid"}`),
	}

	ctrl := gomock.NewController(t)
	notifier := mocks.NewMocknotifier(ctrl)
	notifier.EXPECT().
		NotifyOrderStatusChanged(gomock.Any(), int64(900001), int64(42), "paid").
		Return(nil)

	cfg := &config.Config{}
	cfg.Kafka.RetryBackoff = time.Nanosecond

	ok, processed := processMessageWithoutCommitWithDLQProducer(
		context.Background(),
		zap.NewNop(),
		cfg,
		nil,
		nil,
		"bootstrap.servers",
		notifier,
		message,
	)

	require.True(t, ok)
	require.True(t, processed)
}

func TestProcessInvalidMessageWithoutDLQProducer(t *testing.T) {
	t.Parallel()

	topic := "order_status_notifications"
	ctrl := gomock.NewController(t)
	cfg := &config.Config{}
	cfg.Kafka.DLQTopic = "order_status_notifications_dlq"
	cfg.Kafka.DLQTimeout = time.Second

	ok, processed := processMessageWithoutCommitWithDLQProducer(
		context.Background(),
		zap.NewNop(),
		cfg,
		nil,
		nil,
		"bootstrap.servers",
		mocks.NewMocknotifier(ctrl),
		&ckafka.Message{
			TopicPartition: ckafka.TopicPartition{
				Topic:     &topic,
				Partition: 1,
				Offset:    10,
			},
			Value: []byte(`{bad-json`),
		},
	)

	require.True(t, ok)
	require.False(t, processed)
}

func TestProcessInvalidMessageFailsWhenDLQProducerCreationFails(t *testing.T) {
	t.Parallel()

	topic := "order_status_notifications"
	ctrl := gomock.NewController(t)
	cfg := &config.Config{}
	cfg.Kafka.DLQTopic = "order_status_notifications_dlq"
	cfg.Kafka.DLQTimeout = time.Second

	var producer *ckafka.Producer
	ok, processed := processMessageWithoutCommitWithDLQProducer(
		context.Background(),
		zap.NewNop(),
		cfg,
		&producer,
		nil,
		"",
		mocks.NewMocknotifier(ctrl),
		&ckafka.Message{
			TopicPartition: ckafka.TopicPartition{
				Topic:     &topic,
				Partition: 1,
				Offset:    10,
			},
			Value: []byte(`{bad-json`),
		},
	)

	require.False(t, ok)
	require.False(t, processed)
}

func TestSendToDLQ(t *testing.T) {
	t.Parallel()

	topic := "order_status_notifications"
	message := &ckafka.Message{
		TopicPartition: ckafka.TopicPartition{
			Topic:     &topic,
			Partition: 1,
			Offset:    10,
		},
		Key:   []byte("42"),
		Value: []byte(`{bad-json`),
	}

	ctrl := gomock.NewController(t)
	producer := mocks.NewMockdlqWriter(ctrl)
	cfg := &config.Config{}
	cfg.Kafka.DLQTopic = "order_status_notifications_dlq"
	cfg.Kafka.DLQTimeout = time.Second

	var dlqMessage *ckafka.Message
	producer.EXPECT().
		Produce(gomock.Any(), gomock.Any()).
		DoAndReturn(func(msg *ckafka.Message, deliveryChan chan ckafka.Event) error {
			dlqMessage = msg
			deliveryChan <- &ckafka.Message{}
			return nil
		})

	err := sendToDLQ(context.Background(), cfg, producer, message, errors.New("bad json"))

	require.NoError(t, err)
	require.NotNil(t, dlqMessage)
	require.Equal(t, cfg.Kafka.DLQTopic, *dlqMessage.TopicPartition.Topic)
	require.Equal(t, ckafka.PartitionAny, dlqMessage.TopicPartition.Partition)
	require.Equal(t, message.Key, dlqMessage.Key)
	require.Equal(t, message.Value, dlqMessage.Value)
	require.Contains(t, dlqMessage.Headers, ckafka.Header{
		Key:   "x-original-topic",
		Value: []byte(topic),
	})
	require.Contains(t, dlqMessage.Headers, ckafka.Header{
		Key:   "x-original-partition",
		Value: []byte("1"),
	})
	require.Contains(t, dlqMessage.Headers, ckafka.Header{
		Key:   "x-original-offset",
		Value: []byte("10"),
	})
}

func TestProcessInvalidMessageFailsWhenDLQFails(t *testing.T) {
	t.Parallel()

	topic := "order_status_notifications"
	cfg := &config.Config{}
	cfg.Kafka.DLQTopic = "order_status_notifications_dlq"
	cfg.Kafka.DLQTimeout = time.Second

	ctrl := gomock.NewController(t)
	producer := mocks.NewMockdlqWriter(ctrl)
	producer.EXPECT().
		Produce(gomock.Any(), gomock.Any()).
		Return(errors.New("produce failed"))

	err := sendToDLQ(context.Background(), cfg, producer, &ckafka.Message{
		TopicPartition: ckafka.TopicPartition{
			Topic:     &topic,
			Partition: 1,
			Offset:    10,
		},
		Value: []byte(`{bad-json`),
	}, errors.New("bad json"))

	require.Error(t, err)
}

func TestSendToDLQReturnsDeliveryError(t *testing.T) {
	t.Parallel()

	topic := "order_status_notifications"
	deliveryErr := errors.New("delivery failed")

	ctrl := gomock.NewController(t)
	producer := mocks.NewMockdlqWriter(ctrl)
	producer.EXPECT().
		Produce(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ *ckafka.Message, deliveryChan chan ckafka.Event) error {
			deliveryChan <- &ckafka.Message{
				TopicPartition: ckafka.TopicPartition{
					Error: deliveryErr,
				},
			}
			return nil
		})

	cfg := &config.Config{}
	cfg.Kafka.DLQTopic = "order_status_notifications_dlq"
	cfg.Kafka.DLQTimeout = time.Second

	err := sendToDLQ(context.Background(), cfg, producer, &ckafka.Message{
		TopicPartition: ckafka.TopicPartition{
			Topic:     &topic,
			Partition: 1,
			Offset:    10,
		},
		Value: []byte(`{bad-json`),
	}, errors.New("bad json"))

	require.ErrorIs(t, err, deliveryErr)
}

func TestSendToDLQReturnsEmptyTopicError(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}

	err := sendToDLQ(context.Background(), cfg, nil, &ckafka.Message{}, errors.New("bad json"))

	require.Error(t, err)
}

func TestSendToDLQReturnsTimeout(t *testing.T) {
	t.Parallel()

	topic := "order_status_notifications"

	ctrl := gomock.NewController(t)
	producer := mocks.NewMockdlqWriter(ctrl)
	producer.EXPECT().
		Produce(gomock.Any(), gomock.Any()).
		Return(nil)

	cfg := &config.Config{}
	cfg.Kafka.DLQTopic = "order_status_notifications_dlq"
	cfg.Kafka.DLQTimeout = time.Nanosecond

	err := sendToDLQ(context.Background(), cfg, producer, &ckafka.Message{
		TopicPartition: ckafka.TopicPartition{
			Topic:     &topic,
			Partition: 1,
			Offset:    10,
		},
		Value: []byte(`{bad-json`),
	}, errors.New("bad json"))

	require.Error(t, err)
}

func TestPollBatch(t *testing.T) {
	t.Parallel()

	topic := "order_status_notifications"
	messages := []*ckafka.Message{
		{
			TopicPartition: ckafka.TopicPartition{
				Topic:     &topic,
				Partition: 0,
				Offset:    1,
			},
			Value: []byte(`{"user_id":1,"order_id":1,"status":"paid"}`),
		},
		{
			TopicPartition: ckafka.TopicPartition{
				Topic:     &topic,
				Partition: 0,
				Offset:    2,
			},
			Value: []byte(`{"user_id":1,"order_id":2,"status":"paid"}`),
		},
	}

	ctrl := gomock.NewController(t)
	consumer := mocks.NewMockbatchConsumer(ctrl)
	gomock.InOrder(
		consumer.EXPECT().Poll(1000).Return(messages[0]),
		consumer.EXPECT().Poll(0).Return(messages[1]),
		consumer.EXPECT().Poll(0).Return(nil),
	)

	cfg := &config.Config{}
	cfg.Kafka.BatchSize = 10
	cfg.Kafka.PollTimeoutMs = 1000

	batch, ok := pollBatch(context.Background(), zap.NewNop(), cfg, consumer)

	require.True(t, ok)
	require.Len(t, batch, 2)
	require.Equal(t, messages, batch)
}

func TestPollBatchHandlesPartitionEvents(t *testing.T) {
	t.Parallel()

	topic := "order_status_notifications"
	partitions := []ckafka.TopicPartition{
		{Topic: &topic, Partition: 0},
	}

	ctrl := gomock.NewController(t)
	consumer := mocks.NewMockbatchConsumer(ctrl)
	gomock.InOrder(
		consumer.EXPECT().Poll(1000).Return(ckafka.AssignedPartitions{Partitions: partitions}),
		consumer.EXPECT().Assign(partitions).Return(nil),
		consumer.EXPECT().Poll(1000).Return(ckafka.RevokedPartitions{Partitions: partitions}),
		consumer.EXPECT().Unassign().Return(nil),
		consumer.EXPECT().Poll(1000).Return(nil),
	)

	cfg := &config.Config{}
	cfg.Kafka.BatchSize = 10
	cfg.Kafka.PollTimeoutMs = 1000

	batch, ok := pollBatch(context.Background(), zap.NewNop(), cfg, consumer)

	require.True(t, ok)
	require.Empty(t, batch)
}

func TestPollBatchStopsOnContextCancel(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	consumer := mocks.NewMockbatchConsumer(ctrl)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg := &config.Config{}
	cfg.Kafka.BatchSize = 10
	cfg.Kafka.PollTimeoutMs = 1000

	batch, ok := pollBatch(ctx, zap.NewNop(), cfg, consumer)

	require.False(t, ok)
	require.Empty(t, batch)
}

func TestProcessBatchWithoutCommit(t *testing.T) {
	t.Parallel()

	topic := "order_status_notifications"
	batch := []*ckafka.Message{
		{
			TopicPartition: ckafka.TopicPartition{
				Topic:     &topic,
				Partition: 0,
				Offset:    1,
			},
			Value: []byte(`{"user_id":900001,"order_id":1,"status":"paid"}`),
		},
		{
			TopicPartition: ckafka.TopicPartition{
				Topic:     &topic,
				Partition: 0,
				Offset:    2,
			},
			Value: []byte(`{"user_id":900001,"order_id":2,"status":"cancelled"}`),
		},
	}

	ctrl := gomock.NewController(t)
	notifier := mocks.NewMocknotifier(ctrl)
	gomock.InOrder(
		notifier.EXPECT().
			NotifyOrderStatusChanged(gomock.Any(), int64(900001), int64(1), "paid").
			Return(nil),
		notifier.EXPECT().
			NotifyOrderStatusChanged(gomock.Any(), int64(900001), int64(2), "cancelled").
			Return(nil),
	)

	cfg := &config.Config{}
	cfg.Kafka.RetryBackoff = time.Nanosecond

	ok := processBatchWithoutCommit(context.Background(), zap.NewNop(), cfg, nil, nil, "bootstrap.servers", notifier, batch)

	require.True(t, ok)
}

func TestProcessBatchWithoutCommitStopsOnError(t *testing.T) {
	t.Parallel()

	topic := "order_status_notifications"
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ctrl := gomock.NewController(t)
	notifier := mocks.NewMocknotifier(ctrl)
	notifier.EXPECT().
		NotifyOrderStatusChanged(gomock.Any(), int64(900001), int64(1), "paid").
		Return(errors.New("notify failed"))

	cfg := &config.Config{}
	cfg.Kafka.RetryBackoff = time.Nanosecond

	ok := processBatchWithoutCommit(ctx, zap.NewNop(), cfg, nil, nil, "bootstrap.servers", notifier, []*ckafka.Message{
		{
			TopicPartition: ckafka.TopicPartition{
				Topic:     &topic,
				Partition: 0,
				Offset:    1,
			},
			Value: []byte(`{"user_id":900001,"order_id":1,"status":"paid"}`),
		},
	})

	require.False(t, ok)
}

func TestCommitOffsets(t *testing.T) {
	t.Parallel()

	topic := "order_status_notifications"
	offsets := commitOffsets([]*ckafka.Message{
		{
			TopicPartition: ckafka.TopicPartition{
				Topic:     &topic,
				Partition: 0,
				Offset:    10,
			},
		},
		{
			TopicPartition: ckafka.TopicPartition{
				Topic:     &topic,
				Partition: 0,
				Offset:    12,
			},
		},
		{
			TopicPartition: ckafka.TopicPartition{
				Topic:     &topic,
				Partition: 1,
				Offset:    3,
			},
		},
	})

	got := make(map[string]ckafka.Offset, len(offsets))
	for _, offset := range offsets {
		got[fmt.Sprintf("%s:%d", *offset.Topic, offset.Partition)] = offset.Offset
	}

	require.Equal(t, map[string]ckafka.Offset{
		"order_status_notifications:0": 13,
		"order_status_notifications:1": 4,
	}, got)
}

func TestHelpers(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	cfg.Kafka.BatchSize = 0
	require.Equal(t, 1, batchSize(cfg))

	cfg.Kafka.BatchSize = 25
	require.Equal(t, 25, batchSize(cfg))

	require.Equal(t, "kafka-1:9092,kafka-2:9092", joinBrokers([]string{"kafka-1:9092", "kafka-2:9092"}))
	require.Empty(t, topicName(&ckafka.Message{}))

	_, err := getDLQProducer(nil, nil, "bootstrap.servers")
	require.ErrorIs(t, err, errDLQProducerDisabled)

	producer := &ckafka.Producer{}
	got, err := getDLQProducer(&producer, nil, "bootstrap.servers")
	require.NoError(t, err)
	require.Same(t, producer, got)
}
