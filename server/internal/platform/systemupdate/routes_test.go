package systemupdate

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandlerReturnsVersionStatus(t *testing.T) {
	router := newTestRouter(newTestService(t), allowRequest, allowRequest)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/system/version", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	body := decodeResponse(t, response)
	if body.Code != http.StatusOK || body.Data.CurrentVersion != "0.0.7" || body.Data.CurrentRevision != "revision-a" {
		t.Fatalf("unexpected response: %#v", body)
	}
}

func TestHandlerQueuesCheckRequest(t *testing.T) {
	service := newTestService(t)
	router := newTestRouter(service, allowRequest, allowRequest)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/system/version/check", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusAccepted)
	}
	body := decodeResponse(t, response)
	if body.Code != http.StatusOK || body.Data.State != "check_queued" || body.Data.Action != "check" {
		t.Fatalf("unexpected response: %#v", body)
	}
}

func TestHandlerRequiresCheckBeforeUpdate(t *testing.T) {
	router := newTestRouter(newTestService(t), allowRequest, allowRequest)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/system/version/update", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
	body := decodeResponse(t, response)
	if body.Code != http.StatusConflict || body.Msg != "请先检查是否有可用更新" {
		t.Fatalf("unexpected response: %#v", body)
	}
}

func TestHandlerRunsPermissionMiddleware(t *testing.T) {
	router := newTestRouter(newTestService(t), denyRequest, allowRequest)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/system/version", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

type handlerResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data Status `json:"data"`
}

func newTestRouter(service *Service, read, edit gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(service).Register(router.Group("/api/v1/system"), read, edit)
	return router
}

func decodeResponse(t *testing.T, response *httptest.ResponseRecorder) handlerResponse {
	t.Helper()
	var body handlerResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return body
}

func allowRequest(c *gin.Context) {
	c.Next()
}

func denyRequest(c *gin.Context) {
	c.AbortWithStatus(http.StatusForbidden)
}
