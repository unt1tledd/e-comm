package notifications

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/entity"
	notificationspb "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/notifications/api/v1"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNotificationsServer_SendOrderStatusChangedNotification(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		callbackAddr string
		handler      http.HandlerFunc
		wantErr      bool
	}{
		{
			name:         "empty callback addr",
			callbackAddr: "",
			handler:      nil,
			wantErr:      false,
		},
		{
			name:         "success",
			callbackAddr: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodPost, r.Method)

				var payload entity.CallbackPayload
				err := json.NewDecoder(r.Body).Decode(&payload)
				require.NoError(t, err)

				require.Equal(t, int64(123), payload.UserID)
				require.Equal(t, int64(456), payload.OrderID)
				require.Equal(t, entity.OrderStatusPaid, payload.Status)

				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
		{
			name:         "callback addr without scheme",
			callbackAddr: "without_scheme",
			handler: func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodPost, r.Method)

				var payload entity.CallbackPayload
				err := json.NewDecoder(r.Body).Decode(&payload)
				require.NoError(t, err)

				require.Equal(t, int64(123), payload.UserID)
				require.Equal(t, int64(456), payload.OrderID)
				require.Equal(t, entity.OrderStatusPaid, payload.Status)

				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
		{
			name:         "callback returns error status",
			callbackAddr: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
		{
			name:         "bad callback url",
			callbackAddr: "://bad-url",
			handler:      nil,
			wantErr:      true,
		},
		{
			name:         "callback request failed",
			callbackAddr: "http://127.0.0.1:1",
			handler:      nil,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			callbackAddr := tt.callbackAddr

			if tt.handler != nil {
				server := httptest.NewServer(tt.handler)
				t.Cleanup(server.Close)

				callbackAddr = server.URL

				if tt.name == "callback addr without scheme" {
					callbackAddr = strings.TrimPrefix(server.URL, "http://")
				}
			}

			s := NewNotificationsServer(zap.NewNop(), callbackAddr)

			req := &notificationspb.OrderStatusChangedNotificationRequest{
				UserId:  123,
				OrderId: 456,
				Status:  notificationspb.OrderStatus_ORDER_STATUS_PAID,
			}

			resp, err := s.SendOrderStatusChangedNotification(context.Background(), req)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, resp)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
		})
	}
}
