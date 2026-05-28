package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/config"
	"go.uber.org/zap"
)

func NewOrderStatusRestNotifier(logger *zap.Logger, cfg *config.Config) (*orderStatusRestNotifierImpl, error) {
	return &orderStatusRestNotifierImpl{
		logger: logger,
		cfg:    cfg,
	}, nil
}

type orderStatusRestNotifierImpl struct {
	logger *zap.Logger
	cfg    *config.Config
}

type orderStatusPayload struct {
	UserID  int64  `json:"user_id"`
	OrderID int64  `json:"order_id"`
	Status  string `json:"status"`
}

func (n *orderStatusRestNotifierImpl) NotifyOrderStatusChanged(
	ctx context.Context,
	userID,
	orderID int64,
	status string,
) error {
	callbackAddr := strings.TrimSpace(n.cfg.Clients.CallbackAddr)
	n.logger.Info("Order status changed",
		zap.Int64("user_id", userID),
		zap.Int64("order_id", orderID),
		zap.String("status", status),
	)

	if callbackAddr == "" {
		return nil
	}

	if !strings.HasPrefix(callbackAddr, "http://") && !strings.HasPrefix(callbackAddr, "https://") {
		callbackAddr = "http://" + callbackAddr
	}

	payload := orderStatusPayload{
		UserID:  userID,
		OrderID: orderID,
		Status:  status,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		n.logger.Error("failed to marshal callback payload", zap.Error(err))
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, callbackAddr, bytes.NewReader(body))
	if err != nil {
		n.logger.Error("failed to create callback request",
			zap.String("callback_addr", callbackAddr),
			zap.Error(err),
		)
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: n.cfg.Clients.CallbackTimeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		n.logger.Error("failed to send callback",
			zap.String("callback_addr", callbackAddr),
			zap.Int64("order_id", orderID),
			zap.Error(err),
		)
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		n.logger.Error("callback returned non-2xx status",
			zap.String("callback_addr", callbackAddr),
			zap.Int("status_code", resp.StatusCode),
			zap.Int64("order_id", orderID),
		)
		return fmt.Errorf("callback returned status %d", resp.StatusCode)
	}

	return nil
}
