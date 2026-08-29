package mail

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBeginTwoFactorDoesNotCreatePendingWhenDeviceRequestFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || !strings.HasSuffix(r.URL.Path, "/verify/trusteddevice/securitycode") {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"secret upstream response"}`))
	}))
	defer server.Close()

	client := NewAppleAuthClient()
	client.httpClient = appleTestHTTPClient(server)
	session := &appleAuthSession{
		Channel:         AppleChannelICloudWeb,
		Endpoints:       appleWebAuthEndpoints(RegionInternational),
		ClientID:        appleWebOAuthClientID,
		FrameID:         "frame",
		UserAgent:       appleAuthUserAgent,
		TwoFactorMethod: AppleTwoFactorDevice,
	}

	_, err := client.beginTwoFactor(context.Background(), session)
	var protocol *AppleProtocolError
	if !errors.As(err, &protocol) || protocol.Code != "APPLE_2FA_SERVICE_UNAVAILABLE" || protocol.HTTPStatus != http.StatusBadGateway {
		t.Fatalf("unexpected error: %#v", err)
	}
	if strings.Contains(err.Error(), "secret upstream response") {
		t.Fatalf("upstream response leaked: %v", err)
	}
	client.pendingMu.Lock()
	pendingCount := len(client.pending)
	client.pendingMu.Unlock()
	if pendingCount != 0 {
		t.Fatalf("created %d pending login(s) after challenge request failure", pendingCount)
	}
}

func TestBeginTwoFactorCreatesPendingAfterDeviceRequestSucceeds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || !strings.HasSuffix(r.URL.Path, "/verify/trusteddevice/securitycode") {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewAppleAuthClient()
	client.httpClient = appleTestHTTPClient(server)
	session := &appleAuthSession{
		Channel:         AppleChannelICloudWeb,
		Endpoints:       appleWebAuthEndpoints(RegionInternational),
		ClientID:        appleWebOAuthClientID,
		FrameID:         "frame",
		UserAgent:       appleAuthUserAgent,
		TwoFactorMethod: AppleTwoFactorDevice,
		AppleID:         "owner@example.com",
	}

	result, err := client.beginTwoFactor(context.Background(), session)
	if err != nil || !result.Needs2FA || result.PendingID == "" || result.Message != "Apple 已向受信任设备发送验证码" {
		t.Fatalf("unexpected result: %#v, error: %v", result, err)
	}
	if _, ok := client.getPending(result.PendingID); !ok {
		t.Fatalf("pending login was not stored")
	}
}

func TestValidateDeviceCodeReportsSafeFailureCategories(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		code       string
		httpStatus int
	}{
		{name: "invalid", status: http.StatusBadRequest, code: "APPLE_2FA_FAILED", httpStatus: http.StatusBadRequest},
		{name: "expired", status: http.StatusUnauthorized, code: "APPLE_2FA_SESSION_EXPIRED", httpStatus: http.StatusBadRequest},
		{name: "limited", status: http.StatusTooManyRequests, code: "APPLE_2FA_RATE_LIMITED", httpStatus: http.StatusTooManyRequests},
		{name: "upstream", status: http.StatusServiceUnavailable, code: "APPLE_2FA_SERVICE_UNAVAILABLE", httpStatus: http.StatusBadGateway},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/verify/trusteddevice/securitycode") {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(`{"reason":"secret upstream response"}`))
			}))
			defer server.Close()

			client := NewAppleAuthClient()
			client.httpClient = appleTestHTTPClient(server)
			session := &appleAuthSession{
				Channel:   AppleChannelICloudWeb,
				Endpoints: appleWebAuthEndpoints(RegionInternational),
				ClientID:  appleWebOAuthClientID,
				FrameID:   "frame",
				UserAgent: appleAuthUserAgent,
			}
			err := client.validateDeviceCode(context.Background(), session, "123456")
			var protocol *AppleProtocolError
			if !errors.As(err, &protocol) || protocol.Code != tt.code || protocol.HTTPStatus != tt.httpStatus {
				t.Fatalf("unexpected error: %#v", err)
			}
			if strings.Contains(err.Error(), "secret upstream response") {
				t.Fatalf("upstream response leaked: %v", err)
			}
		})
	}
}

func TestValidateDeviceCodeAcceptsSuccessfulResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/verify/trusteddevice/securitycode") {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewAppleAuthClient()
	client.httpClient = appleTestHTTPClient(server)
	session := &appleAuthSession{
		Channel:   AppleChannelICloudWeb,
		Endpoints: appleWebAuthEndpoints(RegionInternational),
		ClientID:  appleWebOAuthClientID,
		FrameID:   "frame",
		UserAgent: appleAuthUserAgent,
	}
	if err := client.validateDeviceCode(context.Background(), session, "123456"); err != nil {
		t.Fatalf("successful verification returned error: %v", err)
	}
}

func TestICloudWebTwoFactorHeadersIncludeAppID(t *testing.T) {
	session := &appleAuthSession{
		Channel:   AppleChannelICloudWeb,
		Endpoints: appleWebAuthEndpoints(RegionInternational),
		ClientID:  appleWebOAuthClientID,
		FrameID:   "frame",
		UserAgent: appleAuthUserAgent,
	}
	headers := session.twoFactorHeaders()
	if headers["X-Apple-App-Id"] != appleWebOAuthClientID {
		t.Fatalf("X-Apple-App-Id = %q, want %q", headers["X-Apple-App-Id"], appleWebOAuthClientID)
	}
}
