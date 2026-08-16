package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"gokeep/server/internal/common"
)

type handlerResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

type captchaResponse struct {
	CaptchaEnabled bool   `json:"captchaEnabled"`
	Img            string `json:"img"`
	UUID           string `json:"uuid"`
	Expr           string `json:"expr"`
}

func newAuthHandlerRouter(e *authTestEnv) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	authz := common.NewAuthz(e.client, e.rdb, testJWTSecret)
	NewHandler(e.svc, authz).Register(router.Group("/api/v1/auth", authz.Middleware()))
	return router
}

func decodeHandlerResponse(t *testing.T, res *httptest.ResponseRecorder) handlerResponse {
	t.Helper()
	var body handlerResponse
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response %q: %v", res.Body.String(), err)
	}
	return body
}

func TestCaptchaHandlerReturnsDisabledWithoutGeneratingCaptcha(t *testing.T) {
	e := newAuthTestEnv(t)
	e.setConfig(t, "sys.account.captchaEnabled", "false")

	res := httptest.NewRecorder()
	newAuthHandlerRouter(e).ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/auth/captcha", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", res.Code, res.Body.String())
	}
	body := decodeHandlerResponse(t, res)
	var data captchaResponse
	if err := json.Unmarshal(body.Data, &data); err != nil {
		t.Fatalf("decode captcha data: %v", err)
	}
	if body.Code != http.StatusOK || data.CaptchaEnabled || data.Img != "" || data.UUID != "" || data.Expr != "" {
		t.Fatalf("response = %+v, data = %+v; want disabled empty captcha", body, data)
	}
	if keys := e.redis.Keys(); len(keys) != 0 {
		t.Fatalf("Redis keys = %v, want no generated captcha", keys)
	}
}

func TestCaptchaHandlerGeneratesCaptchaWhenEnabled(t *testing.T) {
	e := newAuthTestEnv(t)
	e.setConfig(t, "sys.account.captchaEnabled", "true")

	res := httptest.NewRecorder()
	newAuthHandlerRouter(e).ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/auth/captcha", nil))
	body := decodeHandlerResponse(t, res)
	var data captchaResponse
	if err := json.Unmarshal(body.Data, &data); err != nil {
		t.Fatalf("decode captcha data: %v", err)
	}
	if res.Code != http.StatusOK || body.Code != http.StatusOK || !data.CaptchaEnabled || data.Img == "" || data.UUID == "" || data.Expr == "" {
		t.Fatalf("status = %d, response = %+v, data = %+v; want generated captcha", res.Code, body, data)
	}
	if !e.redis.Exists(common.KeyCaptchaPrefix + data.UUID) {
		t.Fatal("generated captcha was not stored in Redis")
	}
}

func TestLoginHandlerRequiresCaptchaWhenEnabled(t *testing.T) {
	e := newAuthTestEnv(t)
	e.setConfig(t, "sys.account.captchaEnabled", "true")
	user := e.createUser(t, "captcha-required@example.com", "0")
	payload, err := json.Marshal(map[string]string{"username": user.Username, "password": testPassword})
	if err != nil {
		t.Fatalf("marshal login request: %v", err)
	}

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	newAuthHandlerRouter(e).ServeHTTP(res, req)
	body := decodeHandlerResponse(t, res)
	if res.Code != http.StatusOK || body.Code != http.StatusBadRequest || body.Msg != "请输入验证码" {
		t.Fatalf("status = %d, response = %+v; want captcha validation error", res.Code, body)
	}
}

func TestLoginHandlerAllowsEmptyCaptchaWhenDisabled(t *testing.T) {
	e := newAuthTestEnv(t)
	e.setConfig(t, "sys.account.captchaEnabled", "false")
	user := e.createUser(t, "no-captcha@example.com", "0")
	payload, err := json.Marshal(map[string]string{"username": user.Username, "password": testPassword})
	if err != nil {
		t.Fatalf("marshal login request: %v", err)
	}

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	newAuthHandlerRouter(e).ServeHTTP(res, req)
	body := decodeHandlerResponse(t, res)
	if res.Code != http.StatusOK || body.Code != http.StatusOK {
		t.Fatalf("status = %d, response = %+v; want successful login", res.Code, body)
	}
	var data struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body.Data, &data); err != nil || data.Token == "" {
		t.Fatalf("login data = %s, err = %v; want token", body.Data, err)
	}
}
