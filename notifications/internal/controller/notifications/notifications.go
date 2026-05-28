package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/controller/converter"
	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/entity"
	notifications "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/notifications/api/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type notificationsServer struct {
	logger       *zap.Logger
	callbackAddr string
}

func NewNotificationsServer(logger *zap.Logger, callbackAddr string) *notificationsServer {
	return &notificationsServer{
		logger:       logger,
		callbackAddr: callbackAddr,
	}
}

const timeout = 5 * time.Second

func (n *notificationsServer) SendOrderStatusChangedNotification(
	ctx context.Context,
	req *notifications.OrderStatusChangedNotificationRequest,
) (*emptypb.Empty, error) {
	callbackAddr := strings.TrimSpace(n.callbackAddr)

	if callbackAddr == "" {
		return &emptypb.Empty{}, nil
	}
	if !strings.Contains(callbackAddr, "://") {
		callbackAddr = "http://" + callbackAddr
	}

	payload := entity.CallbackPayload{
		UserID:  req.GetUserId(),
		OrderID: req.GetOrderId(),
		Status:  converter.ToOrderStatus(req.GetStatus()),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		n.logger.Error("marshal callback payload failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "marshal callback payload failed")
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		callbackAddr,
		bytes.NewReader(body),
	)

	if err != nil {
		n.logger.Error("create callback request failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "create callback request failed")
	}

	client := &http.Client{
		Timeout: timeout,
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		n.logger.Error("callback request failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "callback request failed")
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		n.logger.Error("callback returned unsuccessful status", zap.Int("status_code", resp.StatusCode))
		return nil, status.Error(codes.Internal, "callback request failed")
	}

	n.logger.Info("Order status changed", zap.String("status", req.GetStatus().String()))
	return &emptypb.Empty{}, nil
}
