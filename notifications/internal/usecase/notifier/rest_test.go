package notifier

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/config"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestOrderStatusRestNotifier_NotifyOrderStatusChanged(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		callbackAddr string
		handler      http.HandlerFunc
		wantErr      bool
	}{
		{
			name: "empty callback addr",
		},
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, "application/json", r.Header.Get("Content-Type"))

				var payload orderStatusPayload
				err := json.NewDecoder(r.Body).Decode(&payload)
				require.NoError(t, err)

				require.Equal(t, int64(900001), payload.UserID)
				require.Equal(t, int64(42), payload.OrderID)
				require.Equal(t, "awaiting_payment", payload.Status)

				w.WriteHeader(http.StatusOK)
			},
		},
		{
			name: "callback addr without scheme",
			handler: func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodPost, r.Method)
				w.WriteHeader(http.StatusNoContent)
			},
		},
		{
			name: "callback returns error status",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			},
			wantErr: true,
		},
		{
			name:         "bad callback url",
			callbackAddr: "://bad-url",
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
					callbackAddr = strings.TrimPrefix(callbackAddr, "http://")
				}
			}

			cfg := &config.Config{}
			cfg.Clients.CallbackAddr = callbackAddr
			cfg.Clients.CallbackTimeout = 5 * time.Second

			notifier, err := NewOrderStatusRestNotifier(zap.NewNop(), cfg)
			require.NoError(t, err)

			err = notifier.NotifyOrderStatusChanged(
				t.Context(),
				900001,
				42,
				"awaiting_payment",
			)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
