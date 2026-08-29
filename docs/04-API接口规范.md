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
| `/api/v1/mail/*` | 邮件账号、隐藏邮箱、收件箱、Session 和活动日志管理（需 JWT 与邮件权限） |
| `/share/v1/*` | 公开分享链接读取与会话交换；分享链接格式为 `/share/v1/latest?email=...&token=...` |

## 安全

- 登录、注册和验证码接口必须限流。
- 密码、令牌和密钥不得出现在响应或日志中。
- 文件和标识符必须由服务端校验，不得直接拼接 SQL 或文件路径。
- 生产环境仅通过 HTTPS 暴露接口，并限制 CORS 来源。
