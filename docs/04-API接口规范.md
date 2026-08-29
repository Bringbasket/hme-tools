# API 接口规范

## 基本约定

- API 前缀为 `/api/v1`，请求与响应默认使用 JSON。
- 资源路径使用复数名词；`GET` 查询、`POST` 创建、`PUT` 更新、`DELETE` 删除。
- 认证使用 `Authorization: Bearer <token>`。
- 分页参数为 `page` 与 `pageSize`，服务端限制最大页容量。
- 操作接口按权限字符授权，例如 `system:user:list`。

统一响应：

```json
{ "code": 200, "msg": "操作成功", "data": {} }
```

失败时 HTTP 状态码与 `code` 保持一致，服务端错误不向客户端泄露内部细节。

## 路由范围

| 分组 | 能力 |
| --- | --- |
| `/healthz`、`/readyz` | 健康和就绪检查 |
| `/api/v1/ping` | API 链路检查 |
| `/api/v1/auth/*` | 验证码、登录、退出、注册、用户信息、动态路由 |
| `/api/v1/system/users` | 用户管理 |
| `/api/v1/system/roles` | 角色管理 |
| `/api/v1/system/menus` | 菜单管理 |
| `/api/v1/system/dict/*` | 字典管理 |
| `/api/v1/system/configs` | 参数配置 |
| `/api/v1/system/settings` | 系统设置和邮件测试 |
| `/api/v1/system/operlogs` | 操作日志 |
| `/api/v1/system/backup/*` | 数据库备份、恢复与 S3 检查 |
| `/api/v1/system/version` | 当前版本与宿主机更新任务状态（需 `system:config:list`） |
| `/api/v1/system/version/check` | 请求宿主机检查最新发布镜像（需 `system:config:edit`） |
| `/api/v1/system/version/update` | 在检查到新版本后请求宿主机拉取、部署和健康检查（需 `system:config:edit`） |
| `/api/v1/mail/*` | 邮件账号、隐藏邮箱、收件箱、Session 和活动日志管理（需 JWT 与邮件权限） |
| `/share/v1/*` | 公开分享链接读取与会话交换；分享链接格式为 `/share/v1/latest?email=...&token=...` |

## 版本更新

版本更新接口只负责展示状态和将请求写入 `DATA_DIR/system`；它们不会在 Web 容器内执行 Docker
命令。宿主机的 `gokeep-update.path` 监听请求文件，并由 `gokeep-update.service` 拉取镜像、重建应用
容器、等待 `/healthz` 成功或自动回滚。

`GET /api/v1/system/version` 返回当前构建和最近任务状态。`POST /api/v1/system/version/check` 创建
检查请求；只有返回 `updateAvailable: true` 后，`POST /api/v1/system/version/update` 才会被接受。
检查或更新已在进行时接口返回 `409`。版本状态中的 `state` 使用 `idle`、`checking`、
`update_available`、`up_to_date`、`updating`、`restarting`、`success` 或 `error`。

## 安全

- 登录、注册和验证码接口必须限流。
- 密码、令牌和密钥不得出现在响应或日志中。
- 文件和标识符必须由服务端校验，不得直接拼接 SQL 或文件路径。
- 生产环境仅通过 HTTPS 暴露接口，并限制 CORS 来源。

## Apple 二次验证错误

`POST /api/v1/mail/session/apple-login/start` 仅在 Apple 接受验证码发送请求后返回 `pendingId`。
发送请求失败不会创建待验证会话。二次验证接口使用以下安全错误码：

| 错误码 | 含义 | HTTP |
| --- | --- | --- |
| `APPLE_2FA_FAILED` | 验证码被 Apple 拒绝或已过期 | 400 |
| `APPLE_2FA_SESSION_EXPIRED` | Apple 二次验证会话失效，需要重新开始登录 | 400 |
| `APPLE_2FA_RATE_LIMITED` | Apple 请求过于频繁 | 429 |
| `APPLE_2FA_REQUEST_FAILED` | Apple 未接受验证码发送请求 | 502 |
| `APPLE_2FA_SERVICE_UNAVAILABLE` | Apple 二次验证服务或网络暂时不可用 | 502 |

错误响应不会包含 Apple 原始响应、Cookie、密码、Token 或验证码内容。
