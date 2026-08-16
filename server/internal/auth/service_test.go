package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	_ "modernc.org/sqlite"

	"gokeep/server/internal/common"
	"gokeep/server/internal/ent"
	"gokeep/server/internal/ent/enttest"
	"gokeep/server/internal/ent/sysuserrole"
)

const (
	testJWTSecret = "auth-test-secret-at-least-32-bytes"
	testPassword  = "correct-horse-battery-staple"
)

type authTestEnv struct {
	client *ent.Client
	rdb    *redis.Client
	redis  *miniredis.Miniredis
	svc    *Service
}

func newAuthTestEnv(t *testing.T) *authTestEnv {
	t.Helper()

	dsn := fmt.Sprintf("file:auth-%d?mode=memory&cache=shared&_pragma=foreign_keys(1)", time.Now().UnixNano())
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	driver := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(driver)))
	t.Cleanup(func() { _ = client.Close() })

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	return &authTestEnv{
		client: client,
		rdb:    rdb,
		redis:  mr,
		svc:    New(client, rdb, testJWTSecret, "development", 2*time.Hour),
	}
}

func (e *authTestEnv) setConfig(t *testing.T, key, value string) {
	t.Helper()
	_, err := e.client.SysConfig.Create().
		SetName(key).
		SetKey(key).
		SetValue(value).
		Save(context.Background())
	if err != nil {
		t.Fatalf("create config %s: %v", key, err)
	}
}

func (e *authTestEnv) createUser(t *testing.T, username, status string) *ent.SysUser {
	t.Helper()
	hash, err := common.HashPassword(testPassword)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user, err := e.client.SysUser.Create().
		SetUsername(username).
		SetPassword(hash).
		SetNickname(username).
		SetStatus(status).
		Save(context.Background())
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func requireBizError(t *testing.T, err error, code int, msg string) {
	t.Helper()
	var biz *common.BizError
	if !errors.As(err, &biz) {
		t.Fatalf("error = %v, want BizError", err)
	}
	if biz.Code != code || biz.Msg != msg {
		t.Fatalf("BizError = (%d, %q), want (%d, %q)", biz.Code, biz.Msg, code, msg)
	}
}

func performAuthorizedRequest(router http.Handler, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	return res
}

func TestLoginIssuesSessionAndLogoutInvalidatesIt(t *testing.T) {
	e := newAuthTestEnv(t)
	e.setConfig(t, "sys.account.captchaEnabled", "false")
	user := e.createUser(t, "member@example.com", "0")

	token, err := e.svc.Login(context.Background(), LoginReq{
		Username: user.Username,
		Password: testPassword,
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	claims, err := common.ParseToken(testJWTSecret, token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if claims.UserID != user.ID || claims.JTI == "" {
		t.Fatalf("claims = %+v, want user %d with JTI", claims, user.ID)
	}
	sessionKey := common.KeySessionPrefix + claims.JTI
	if got, err := e.rdb.Get(context.Background(), sessionKey).Result(); err != nil || got != strconv.FormatInt(user.ID, 10) {
		t.Fatalf("session = %q, %v; want user ID", got, err)
	}
	if ttl := e.redis.TTL(sessionKey); ttl != 2*time.Hour {
		t.Fatalf("session TTL = %v, want 2h", ttl)
	}

	gin.SetMode(gin.TestMode)
	authz := common.NewAuthz(e.client, e.rdb, testJWTSecret)
	router := gin.New()
	router.GET("/protected", authz.Middleware(), func(c *gin.Context) {
		ident := common.CurrentIdentity(c)
		if ident == nil || ident.UserID != user.ID || ident.JTI != claims.JTI {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	})
	if res := performAuthorizedRequest(router, "/protected", token); res.Code != http.StatusNoContent {
		t.Fatalf("protected status = %d, want 204; body=%s", res.Code, res.Body.String())
	}

	e.svc.Logout(context.Background(), &common.Identity{UserID: user.ID, JTI: claims.JTI})
	if e.redis.Exists(sessionKey) {
		t.Fatal("session key still exists after logout")
	}
	if e.redis.Exists(common.KeyPermsPrefix + strconv.FormatInt(user.ID, 10)) {
		t.Fatal("permission cache still exists after logout")
	}
	if res := performAuthorizedRequest(router, "/protected", token); res.Code != http.StatusUnauthorized {
		t.Fatalf("logged-out status = %d, want 401", res.Code)
	}
}

func TestLoginCredentialAndStatusPolicy(t *testing.T) {
	e := newAuthTestEnv(t)
	e.setConfig(t, "sys.account.captchaEnabled", "false")
	e.createUser(t, "active@example.com", "0")
	e.createUser(t, "disabled@example.com", "1")

	_, wrongPassword := e.svc.Login(context.Background(), LoginReq{Username: "active@example.com", Password: "wrong-password"})
	_, unknownUser := e.svc.Login(context.Background(), LoginReq{Username: "missing@example.com", Password: "wrong-password"})
	requireBizError(t, wrongPassword, http.StatusBadRequest, "用户名或密码错误")
	requireBizError(t, unknownUser, http.StatusBadRequest, "用户名或密码错误")

	_, disabled := e.svc.Login(context.Background(), LoginReq{Username: "disabled@example.com", Password: testPassword})
	requireBizError(t, disabled, http.StatusForbidden, "账号已停用，请联系管理员")
}

func TestLoginCaptchaIsSingleUse(t *testing.T) {
	e := newAuthTestEnv(t)
	e.setConfig(t, "sys.account.captchaEnabled", "true")
	user := e.createUser(t, "captcha@example.com", "0")
	ctx := context.Background()
	key := common.KeyCaptchaPrefix + "captcha-id"

	e.redis.Set(key, "12")
	_, err := e.svc.Login(ctx, LoginReq{
		Username: user.Username, Password: testPassword,
		CaptchaUUID: "captcha-id", CaptchaCode: "99",
	})
	requireBizError(t, err, http.StatusBadRequest, "验证码错误")
	if e.redis.Exists(key) {
		t.Fatal("incorrect captcha was not consumed")
	}

	_, err = e.svc.Login(ctx, LoginReq{
		Username: user.Username, Password: testPassword,
		CaptchaUUID: "captcha-id", CaptchaCode: "12",
	})
	requireBizError(t, err, http.StatusBadRequest, "验证码已过期，请刷新后重试")

	e.redis.Set(key, "12")
	if _, err := e.svc.Login(ctx, LoginReq{
		Username: user.Username, Password: testPassword,
		CaptchaUUID: "captcha-id", CaptchaCode: "12",
	}); err != nil {
		t.Fatalf("login with valid captcha: %v", err)
	}
	if e.redis.Exists(key) {
		t.Fatal("successful captcha was not consumed")
	}
}

func TestAuthzRequiresPermissionAndAllowsAdmin(t *testing.T) {
	e := newAuthTestEnv(t)
	ctx := context.Background()
	member := e.createUser(t, "operator@example.com", "0")
	admin := e.createUser(t, "admin@example.com", "0")
	disabledRoleUser := e.createUser(t, "disabled-role@example.com", "0")
	disabledUser := e.createUser(t, "disabled-user@example.com", "1")

	operatorRole := e.client.SysRole.Create().
		SetName("Operator").SetCode("operator").SetStatus("0").SaveX(ctx)
	adminRole := e.client.SysRole.Create().
		SetName("Administrator").SetCode("admin").SetIsAdmin(true).SetStatus("0").SaveX(ctx)
	disabledRole := e.client.SysRole.Create().
		SetName("Disabled Operator").SetCode("disabled-operator").SetStatus("1").SaveX(ctx)
	menu := e.client.SysMenu.Create().
		SetName("Reports").SetPath("/reports").SetMenuType("C").
		SetPerms("system:report:list").SetStatus("0").SaveX(ctx)
	e.client.SysUserRole.Create().SetUserID(member.ID).SetRoleID(operatorRole.ID).SaveX(ctx)
	e.client.SysUserRole.Create().SetUserID(admin.ID).SetRoleID(adminRole.ID).SaveX(ctx)
	e.client.SysUserRole.Create().SetUserID(disabledRoleUser.ID).SetRoleID(disabledRole.ID).SaveX(ctx)
	e.client.SysRoleMenu.Create().SetRoleID(operatorRole.ID).SetMenuID(menu.ID).SaveX(ctx)
	e.client.SysRoleMenu.Create().SetRoleID(disabledRole.ID).SetMenuID(menu.ID).SaveX(ctx)

	memberToken, err := e.svc.issueSession(ctx, member.ID)
	if err != nil {
		t.Fatalf("issue member session: %v", err)
	}
	adminToken, err := e.svc.issueSession(ctx, admin.ID)
	if err != nil {
		t.Fatalf("issue admin session: %v", err)
	}
	disabledRoleToken, err := e.svc.issueSession(ctx, disabledRoleUser.ID)
	if err != nil {
		t.Fatalf("issue disabled-role session: %v", err)
	}
	disabledUserToken, err := e.svc.issueSession(ctx, disabledUser.ID)
	if err != nil {
		t.Fatalf("issue disabled-user session: %v", err)
	}

	gin.SetMode(gin.TestMode)
	authz := common.NewAuthz(e.client, e.rdb, testJWTSecret)
	router := gin.New()
	register := func(path, perm string) {
		router.GET(path, authz.Middleware(), authz.Require(perm), func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})
	}
	register("/reports", "system:report:list")
	register("/users", "system:user:list")

	if res := performAuthorizedRequest(router, "/reports", memberToken); res.Code != http.StatusNoContent {
		t.Fatalf("allowed permission status = %d, want 204", res.Code)
	}
	if res := performAuthorizedRequest(router, "/users", memberToken); res.Code != http.StatusForbidden {
		t.Fatalf("missing permission status = %d, want 403", res.Code)
	}
	if res := performAuthorizedRequest(router, "/users", adminToken); res.Code != http.StatusNoContent {
		t.Fatalf("admin bypass status = %d, want 204", res.Code)
	}
	if res := performAuthorizedRequest(router, "/reports", disabledRoleToken); res.Code != http.StatusForbidden {
		t.Fatalf("disabled role status = %d, want 403", res.Code)
	}
	if res := performAuthorizedRequest(router, "/reports", disabledUserToken); res.Code != http.StatusForbidden {
		t.Fatalf("disabled user status = %d, want 403", res.Code)
	}
}

func TestRegistrationDefaultsClosedAndEmailAttemptsAreLimited(t *testing.T) {
	e := newAuthTestEnv(t)
	ctx := context.Background()

	cfg := e.svc.GetRegisterConfig(ctx)
	if cfg.RegisterEnabled || cfg.EmailCodeEnabled {
		t.Fatalf("default registration config = %+v, want all disabled", cfg)
	}
	_, _, _, _, _, err := e.svc.Register(ctx, RegisterReq{Email: "new@example.com", Password: testPassword})
	requireBizError(t, err, http.StatusForbidden, "注册暂未开放")

	email := "new@example.com"
	key := common.KeyEmailCodePrefix + emailCodePurposeRegister + ":" + email
	record := emailCodeRecord{Code: "123456", ExpiresAt: time.Now().Add(emailCodeTTL)}
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal email code: %v", err)
	}
	e.redis.Set(key, string(raw))
	e.redis.SetTTL(key, emailCodeTTL)

	for attempt := 1; attempt <= emailCodeMaxAttempts; attempt++ {
		err = e.svc.verifyEmailCode(ctx, email, "000000")
		if attempt < emailCodeMaxAttempts {
			remaining := emailCodeMaxAttempts - attempt
			requireBizError(t, err, http.StatusBadRequest, fmt.Sprintf("验证码错误（剩余 %d 次尝试）", remaining))
			continue
		}
		requireBizError(t, err, http.StatusTooManyRequests, "尝试次数过多，请重新获取验证码")
	}
	if e.redis.Exists(key) {
		t.Fatal("email verification code still exists after maximum attempts")
	}
}

func TestRegisterCreatesNormalizedUserRoleAndSession(t *testing.T) {
	e := newAuthTestEnv(t)
	ctx := context.Background()
	e.setConfig(t, "sys.account.registerUser", "true")
	commonRole := e.client.SysRole.Create().
		SetName("Common User").SetCode("common").SetStatus("0").SaveX(ctx)

	token, userID, username, nickname, parentUserID, err := e.svc.Register(ctx, RegisterReq{
		Email:    "  New.User@Example.COM ",
		Password: testPassword,
		Nickname: " ",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if username != "new.user@example.com" || nickname != "new.user" || parentUserID != 0 {
		t.Fatalf("registered user = (%q, %q, parent=%d)", username, nickname, parentUserID)
	}
	user, err := e.client.SysUser.Get(ctx, userID)
	if err != nil {
		t.Fatalf("get registered user: %v", err)
	}
	if user.Email == nil || *user.Email != username {
		t.Fatalf("stored email = %v, want %q", user.Email, username)
	}
	assigned, err := e.client.SysUserRole.Query().
		Where(sysuserrole.UserID(userID), sysuserrole.RoleID(commonRole.ID)).
		Exist(ctx)
	if err != nil || !assigned {
		t.Fatalf("common role assigned = %v, %v", assigned, err)
	}
	claims, err := common.ParseToken(testJWTSecret, token)
	if err != nil || claims.UserID != userID {
		t.Fatalf("registration token claims = %+v, %v", claims, err)
	}
	if !e.redis.Exists(common.KeySessionPrefix + claims.JTI) {
		t.Fatal("registration session was not stored")
	}

	_, _, _, _, _, err = e.svc.Register(ctx, RegisterReq{Email: username, Password: testPassword})
	requireBizError(t, err, http.StatusConflict, "该邮箱已注册，请直接登录")
}

func TestNormalizeEmail(t *testing.T) {
	got, err := normalizeEmail("  User.Name+tag@Example.COM ")
	if err != nil || got != "user.name+tag@example.com" {
		t.Fatalf("normalizeEmail() = %q, %v", got, err)
	}
	for _, value := range []string{"", "missing-at.example.com", "Name <user@example.com>"} {
		if normalized, err := normalizeEmail(value); err == nil {
			t.Fatalf("normalizeEmail(%q) = %q, want error", value, normalized)
		}
	}
}
