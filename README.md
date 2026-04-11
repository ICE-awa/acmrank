# ACMRank

ACMRank 是一个面向华南师范大学校内使用的竞赛档案与训练排行榜平台。

## 平台功能
- 公开展示用户个人页。
- 聚合 `Codeforces`、`AtCoder`、`洛谷` 的过题情况。
- 支持用户聚合视图、平台聚合视图、单账号视图。
- 用户主界面只展示 `AC` 题目列表，不展示完整提交历史。
- 展示 `ICPC` 奖项历史，便于沉淀个人竞赛档案。
- 提供基于 `SCNU Rating` 的训练排行榜。
- 展示 `SCNU Rating` 折线图，用于查看训练趋势变化。
- 展示“每日新 `AC` 题数”热力图，用于查看近期训练活跃度。
- 支持多账号绑定与审核，保证平台账号归属清晰。
- 支持多平台同步与兜底链路，尽量保证数据完整性与稳定性。

## 目标
- 帮助校内成员统一管理个人竞赛档案。
- 更直观地展示近期训练成果与成长趋势。
- 为训练、交流、选拔和回顾提供统一的公开信息入口。

## 仓库结构
- `web/`：React + TypeScript + Vite 前端工程。
- `server/`：Go 后端工程，后续承载 `api / scheduler / aggregator`。
- `sync/`：Python 同步工程，后续承载平台抓取与同步链路。
- `deploy/`：Docker、Docker Compose 与后续部署资产。
- `scripts/`：统一的本地 bootstrap 和检查脚本。

## 本地初始化
- 前置依赖：`Go 1.25`、`Node.js 22`、`pnpm 10`、`Python 3.13`。
- 执行 `make bootstrap` 初始化 Go、Web、Python 三个子工程依赖。
- 执行 `make check` 运行当前骨架的基础验证。
- 所有脚本都会把 `Go / Corepack / pnpm / Playwright / pip` 的缓存落到仓库根目录下的 `.cache/`，避免依赖宿主机默认 `HOME` 写权限。
- `scripts/dev-env.sh` 默认把 Playwright 下载连接超时调到 `120000ms`，适配较慢的网络环境。
- Linux 下运行 Playwright E2E 还需要系统浏览器依赖；当前环境缺少 `libnspr4`，Playwright 额外提示还需要 `libX11-xcb.so.1`、`libasound.so.2`。

## 本地基础设施
- `compose.yaml` 提供本地开发依赖：`PostgreSQL`、`Redis`、`NATS + JetStream`、`ElasticSearch`。
- 默认连接参数写在 `.env.example` 中，后续服务接入时可直接复用这些地址和端口。
- 如果宿主机已有本地数据库或缓存占用了标准端口，可以在 `.env` 中覆盖 `POSTGRES_PORT`、`REDIS_PORT`、`NATS_CLIENT_PORT`、`NATS_MONITOR_PORT`、`ELASTICSEARCH_PORT`。
- `ElasticSearch` 首次拉取镜像体积较大，本地首次 `docker compose up -d` 可能明显慢于其他三个基础服务。
- 启动命令：`docker compose up -d` 或 `make infra-up`
- 停止命令：`docker compose down` 或 `make infra-down`
- 健康检查：`make infra-check`

## 数据库迁移
- 后端 schema 迁移固定使用 `goose`，迁移文件位于 `server/db/migrations/`。
- 执行迁移前先启动本地 `PostgreSQL`，例如 `docker compose up -d postgres` 或 `make infra-up`。
- 迁移命令会先加载仓库根目录的 `.env.example`，再覆盖 `.env`，因此本地端口改动后不需要再手工改命令。
- 查看迁移状态：`make migrate-status`
- 执行全部迁移：`make migrate-up`
- 回滚一版迁移：`make migrate-down`
- 重置并重放迁移：`make migrate-reset`

## 当前 API 鉴权骨架
- 当前 `api` 服务已提供 `v1` 鉴权接口：
  - `POST /api/v1/auth/register`
  - `POST /api/v1/auth/verify-email`
  - `POST /api/v1/auth/login`
  - `POST /api/v1/auth/refresh`
  - `POST /api/v1/auth/logout`
  - `GET /api/v1/users/me`
  - `GET /api/v1/accounts`
  - `POST /api/v1/accounts`
  - `DELETE /api/v1/accounts/:id`
  - `GET /api/v1/admin/platform-accounts`
  - `POST /api/v1/admin/platform-accounts/:id/verify`
  - `POST /api/v1/admin/platform-accounts/:id/disable`
  - `POST /api/v1/admin/platform-accounts/:id/reject`
- 认证模型固定为 `JWT AT + RT + HttpOnly Cookie`。
- 当前仓库尚未接入真实邮件投递能力，因此“邮箱验证基础流程”阶段会直接在注册响应里返回一次性的邮箱验证 token，便于本地联调与自动化测试。
- 本地默认认证配置已写入 `.env.example`，生产环境必须覆盖默认密钥。
- 当前管理端审核接口通过 `ACMRANK_ADMIN_USERNAMES` 控制可访问的站内用户名列表，适合作为角色系统落地前的过渡方案。
