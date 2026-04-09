# Deploy

本目录承载 Docker、Docker Compose 和后续部署资产。

当前已提供本地开发基础设施编排，见仓库根目录 [compose.yaml](../compose.yaml)。

## 服务清单
- `PostgreSQL`：默认端口 `5432`
- `Redis`：默认端口 `6379`
- `NATS + JetStream`：客户端端口 `4222`，监控端口 `8222`
- `ElasticSearch`：默认端口 `9200`

## 初始化资源
- `deploy/postgres/init/01-bootstrap.sql`：初始化 `citext` 和 `pg_trgm` 扩展，供后续用户与搜索功能复用

## 常用命令
- `docker compose up -d`
- `docker compose ps`
- `bash scripts/infra-check.sh`

## 端口覆盖
- 如果宿主机已有服务占用了默认端口，在仓库根目录创建 `.env` 并覆盖对应变量即可，例如 `REDIS_PORT=6380`
