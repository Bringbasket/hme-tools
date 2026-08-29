# GoKeep

GoKeep 是一个前后端分离、可单二进制部署的基础后台管理框架。本仓库首版只保留通用系统能力，不包含具体业务模块。

## 功能

- 登录、验证码、邮箱注册与会话管理
- 用户、角色、菜单和 RBAC 权限控制
- 字典管理、系统设置和操作日志
- PostgreSQL 数据持久化与 Redis 会话缓存
- 数据库备份、S3 存储和定时备份
- 响应式管理后台、暗色主题和动态路由
- 前端静态资源嵌入 Go 二进制

## 技术栈

| 部分 | 技术 |
| --- | --- |
| 前端 | Vue 3、TypeScript、Vite、Tailwind CSS、Pinia |
| 后端 | Go、Gin、Ent |
| 存储 | PostgreSQL 15+、Redis 7+ |
| 工程 | pnpm workspace、Docker Compose、GitHub Actions |

## 目录

```text
apps/web/        Vue 管理后台
server/          Go API、系统服务与内嵌前端
deploy/compose/  本地 PostgreSQL、Redis、MinIO
scripts/         开发与验证脚本
```

## 文档

前端平台外壳、页面骨架、组件语义、状态反馈、分页和移动端验收规则见：

- `docs/01-技术栈规范.md`：依赖基线、主题 token 和工程约定
- `docs/02-总体架构与目录规范.md`：平台外壳、领域边界和目录职责
- `docs/05-前端开发规范.md`：Vue 页面、请求、状态和组件实现
- `docs/06-UI设计规范.md`：管理台布局、视觉层级和交互验收
- `docs/08-移动端适配规范.md`：抽屉侧栏、表格、弹窗和移动端验收

## 本地启动

环境要求：Go 1.25+、Node.js 20+、pnpm 11+、Docker。

```powershell
Copy-Item .env.example .env
docker compose -f deploy/compose/docker-compose.yml up -d
pnpm install
pnpm dev
```

另开一个终端启动后端：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\dev-server.ps1
```

默认地址：

- 管理后台：`http://localhost:5173`
- API：`http://localhost:8080`
- 健康检查：`http://localhost:8080/healthz`

开发环境首次启动会创建管理员账号 `admin / admin123`。登录后应立即修改默认密码。

## 常用命令

```powershell
pnpm build
pnpm typecheck
pnpm test
Set-Location server
go test ./...
go generate ./internal/ent
```

生产环境必须设置强随机 `JWT_SECRET` 和真实 `DATABASE_URL`，并通过反向代理启用 HTTPS。
