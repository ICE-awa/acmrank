# ACMRank 技术架构设计

## 1. 文档约定
- 本文档以 `docs/PRD.md` 为上位产品定义。
- 如果迁移后的代码、历史文档、原型实现与当前产品口径冲突，先修正文档，再继续开发。
- 当前仓库工作目录以 `/opt/acmrank` 为准，不再依赖旧路径上下文。

## 2. 固定技术决策

### 2.1 总体原则
- 单仓库、多进程，不拆微服务。
- `PostgreSQL` 是唯一事实源。
- `Redis` 只承担会话、限流、缓存、短期状态。
- `NATS + JetStream` 只承担异步任务与事件，不作为业务真相存储。
- `ElasticSearch` 只做派生索引和搜索增强，不是主查询源。
- `Python` 只负责访问外部平台和抓取，不承载核心业务规则。
- `first_ac_at` 必须是独立维护的事实字段，排行榜和热力图都基于它计算。
- 存储统一使用 `UTC`，业务窗口与页面展示使用 `Asia/Shanghai`。

### 2.2 固定技术栈
- 前端：`React`、`TypeScript`、`Vite`、`React Router`、`shadcn/ui`、`Tailwind CSS`
- 后端：`Go`、`Gin`、`pgxpool`
- 同步抓取：`Python`、`httpx`、`curl_cffi`
- 基础设施：`PostgreSQL`、`Redis`、`NATS + JetStream`、`ElasticSearch`
- 部署：`Docker`、`Docker Compose`
- 数据库迁移：`goose`

### 2.3 鉴权固定方案
- 后端鉴权固定为 `JWT AT + RT`。
- `AT` 与 `RT` 都放在 `Cookie` 中，不放到前端 `localStorage` 或 `sessionStorage`。
- 刷新策略固定为“主动刷新 + `401` 被动刷新兜底”：
  - 主动刷新：在 `AT` 即将过期前刷新
  - 被动刷新：请求收到 `401` 后尝试刷新并重放一次
- 登出、撤销、轮换等设计都需要围绕 `AT + RT` 方案展开。

## 3. 运行单元

### 3.1 服务划分
| 服务 | 技术 | 责任 |
| --- | --- | --- |
| `web` | React | 用户端和管理端统一前端 |
| `api` | Go + Gin | 鉴权、用户资料、账号、公开页、排行榜、同步入口、扩展接口 |
| `scheduler` | Go | 周期任务、重试、`23:59` 快照、`Clist` 回填投递 |
| `aggregator` | Go | 归并题目事实、维护 `first_ac_at`、计算 `AT / CF / SCNU`、刷新排行榜与索引 |
| `sync-worker` | Python | 调用 `Codeforces / AtCoder / 洛谷 / ICPC / Clist` |
| `atcoder-extension` | 浏览器扩展 | 用户本机最后兜底，同步窗口内 `AC` |

### 3.2 逻辑拓扑
```text
web -> api -> PostgreSQL
             -> Redis
             -> NATS / JetStream

scheduler -> NATS / JetStream
sync-worker -> PostgreSQL / NATS / JetStream
aggregator -> PostgreSQL / ElasticSearch

atcoder-extension -> api -> PostgreSQL / NATS / JetStream
```

## 4. 领域边界

### 4.1 `api`
- 处理注册、登录、会话与权限。
- 提供公开用户页、排行榜、问题列表、奖项历史接口。
- 处理平台账号绑定、审核流转入口和手动同步入口。
- 处理 `AtCoder` 扩展的 `init / upload / complete` 接口。
- 不直接执行长耗时平台同步。

### 4.2 `scheduler`
- 按周期创建平台同步任务。
- 投递 `Clist` 回填任务。
- 在每天 `23:59` 生成排行榜基线快照，用于次日变化值。
- 在每天 `00:00` 开启新的“今日新通过题数”统计窗口。
- 重试失败任务并控制退避。

### 4.3 `sync-worker`
- 访问外部平台与 `Clist`。
- 保存原始响应、来源、时间与错误信息。
- 产出统一的“资料、比赛、AC 记录、奖项、题目元数据”结果。
- 不计算 `SCNU Rating`，也不直接维护排行榜。

### 4.4 `aggregator`
- 维护去重后的平台题目事实。
- 维护 `first_ac_at`。
- 生成平台聚合视图、单账号视图、用户总览。
- 基于 `Clist problem.rating` 计算 `AT_Best / AT_Recent / CF_Best / CF_Recent / AT_Training / CF_Training / SCNU Rating`。
- 刷新排行榜快照和搜索索引。

## 5. 数据职责

### 5.1 PostgreSQL
必须落库并以其为准的数据包括：
- 用户与平台账号归属
- 账号审核状态
- 平台资料快照
- 原始同步结果引用
- `AC` 原始事件
- 去重后的题目事实与 `first_ac_at`
- 比赛 `AC` 汇总
- 奖项历史
- `Clist` 映射与题目权重
- 聚合统计、排行榜、每日快照
- 同步窗口与同步任务状态

### 5.2 Redis
仅用于：
- 鉴权短期状态
- `RT` 轮换、撤销或相关短期状态
- 限流
- 验证码或短期令牌
- 幂等键
- 热门缓存
- 短期锁

### 5.3 ElasticSearch
仅用于派生索引：
- 题目搜索增强
- 管理端日志检索
- 可选的排行榜或用户搜索增强

## 6. 核心数据模型

### 6.1 用户与账号
- `users`
  - `username`
  - `email`
  - `real_name`
- `platform_accounts`
  - `platform`
  - `handle`
  - `site_user_id`
  - `status`
  - `verified_at`

### 6.2 平台事实
- `accepted_event_raw`
  - 原始 `AC` 事件
  - `source`
  - `source_url`
  - `raw_payload_ref`
- `problem_fact`
  - 以“用户 + 平台 + 题目”为粒度
  - 维护 `first_ac_at`
  - 维护题目元数据与 `clist_rating`
- `contest_ac_summary`
  - 以“用户 + 平台 + 比赛”为粒度
  - 维护该场比赛 `AC` 了哪些题
- `profile_snapshot`
  - 平台资料与原生 `rating`
- `award_record`
  - `ICPC` 奖项历史

### 6.3 聚合与快照
- `user_platform_stats`
  - 平台去重过题数
  - 平台展示分数
- `user_rating_snapshot`
  - `AT_Training`
  - `CF_Training`
  - `Recent`
  - `SCNU Rating`
- `leaderboard_snapshot`
  - 每日 `23:59` 基线
  - 排行榜当前视图缓存
- `heatmap_daily_stats`
  - 按日统计“每日新 `AC` 题数”

## 7. 平台同步架构

### 7.1 通用同步流程
1. `scheduler` 投递同步任务。
2. `sync-worker` 读取平台配置并调用外部平台。
3. 原始响应与结构化结果写入 `PostgreSQL`。
4. `aggregator` 归并 `AC` 事实并更新 `first_ac_at`。
5. `aggregator` 回填 `Clist rating`、重算训练分与排行榜。
6. 派生结果同步到 `ElasticSearch`。

### 7.2 Codeforces
- 主链路：官方 API。
- 兜底：`Clist`。
- `AC` 事实以 `user.status verdict = OK` 为优先来源。
- 展示分数取所有已验证账号中的最高 `maxRating`。

### 7.3 AtCoder
- 采用多级兜底，不把用户扩展当主链路。
- 固定优先顺序：
  1. 运营 `Cookie` 主链路
  2. `Clist`
  3. 第三方 API
  4. 用户扩展窗口期 `AC` 同步
- 用户扩展只上传窗口内 `AC`，不是完整提交历史。
- 服务端必须保存来源，并允许后续高可信来源纠正 `first_ac_at`。
- 主链路、`Clist`、第三方 API 任一失效都要主动告警。

### 7.4 洛谷
- 只维护一条稳定链路。
- 输出资料、过题、平台 `rating`。
- 不参与 `SCNU Rating` 计算。

### 7.5 ICPC
- 只维护一条稳定链路。
- 只同步奖项历史，不进入刷题统计。
- 允许人工修正。

## 8. AtCoder 最后兜底链路

### 8.1 设计原则
- 用户扩展不是主链路。
- 扩展只在前 3 层链路不足时承担人工补数作用。
- 扩展只上传窗口内 `AC` 记录。
- 服务端根据这些 `AC` 记录归并：
  - 已通过题列表
  - `first_ac_at`
  - 某场比赛 `AC` 了哪些题

### 8.2 刷新窗口
- 默认窗口：`上次成功同步时间 -> 当前触发时间`
- 首次同步或窗口缺失：由服务端下发初始化窗口
- 窗口状态必须落库，不能只存在浏览器或 `Redis`

## 9. API 边界

### 9.1 公开接口
- `GET /api/v1/public/users/:username`
- `GET /api/v1/public/users/:username/problems`
- `GET /api/v1/public/users/:username/awards`
- `GET /api/v1/public/rankings`

### 9.2 用户接口
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/verify-email`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/logout`
- `GET /api/v1/users/me`
- `GET /api/v1/accounts`
- `POST /api/v1/accounts`
- `DELETE /api/v1/accounts/:id`
- `POST /api/v1/accounts/:id/sync`

### 9.3 管理接口
- `GET /api/v1/admin/platform-accounts`
- `POST /api/v1/admin/platform-accounts/:id/verify`
- `POST /api/v1/admin/platform-accounts/:id/disable`
- `POST /api/v1/admin/platform-accounts/:id/reject`
- `GET /api/v1/admin/sync-jobs`
- `GET /api/v1/admin/config`
- `PUT /api/v1/admin/config`

### 9.4 扩展接口
- `POST /api/v1/extension/atcoder/init`
- `POST /api/v1/extension/atcoder/upload`
- `POST /api/v1/extension/atcoder/complete`

### 9.5 T05 当前落地说明
- 鉴权模型固定为 `JWT AT + RT + Cookie`。
- `AT` 与 `RT` 都由服务端写入 `HttpOnly Cookie`。
- 当前尚未接入真实邮件通道，因此 `register` 响应会直接返回一次性邮箱验证 token，仅作为本地开发和自动化测试阶段的临时方案。

### 9.6 T06 当前落地说明
- 平台账号绑定接口已支持新增、删除、查询以及审核状态流转。
- 平台账号唯一归属目前通过数据库中的 `UNIQUE (platform, handle)` 约束保证。
- 管理端审核接口当前通过配置项 `ACMRANK_ADMIN_USERNAMES` 控制可访问的站内用户名列表，后续如引入专门角色模型再替换。
- 账号列表接口当前支持 `limit / offset` 分页参数，默认页大小为 `50`，最大页大小为 `100`。

## 10. 前端结构

### 10.1 页面
- `/rankings`
- `/u/:username`
- `/u/:username/platform/:platform`
- `/u/:username/platform/:platform/account/:accountId`
- `/settings/accounts`
- `/settings/sync`
- `/admin/*`

### 10.2 页面职责
- 用户页展示：
  - 平台聚合统计
  - `AC` 题目列表
  - `SCNU Rating` 折线图
  - 热力图
  - `ICPC` 奖项历史
- 排行榜页展示：
  - `SCNU Rating`
  - 昨日 `23:59` 变化值
  - 可切换排序键

## 11. 时间与快照
- 每日 `23:59` 生成排行榜基线快照。
- 每日 `00:00` 开启新的“今日新通过题数”统计。
- 热力图按自然日统计“每日新 `AC` 题数”。
- 近半个月、近一个月窗口按 `Asia/Shanghai` 业务日期解释。

## 12. 运维与告警
- `AtCoder` 主链路失效告警
- `AtCoder Clist` 失效告警
- `AtCoder` 第三方 API 失效告警
- 同步任务失败率、延迟、重试次数需要可观测
- 运营 `Cookie`、`Clist` 凭证等敏感信息必须加密存储

## 13. 实现约束
- 不要让 `Redis` 或 `ElasticSearch` 承担事实查询职责。
- 不要把排行榜核心公式塞进 Python worker。
- 不要让用户页展示完整提交历史来替代 `AC` 视图。
- 不要把 `AtCoder` 扩展实现成主同步链路。
