# ACMRank PRD

## 1. 文档约定
- 版本：v0.2
- 更新时间：2026-04-09
- 当前仓库上下文以 `/opt/acmrank` 为准，不再假设旧目录 `/mnt/f/archive/acmrank` 中的文档或实现仍然有效。
- 如果 `docs/` 内文档与“已确认事实”冲突，以“已确认事实”为准，并且必须先修正文档，再继续实现。

## 2. 产品定位
- ACMRank 是华南师范大学校内使用的竞赛档案与训练排行榜系统。
- 用户公开展示个人页，核心内容包括：
  - `Codeforces / AtCoder / 洛谷` 的聚合过题情况
  - 平台聚合视图
  - 单账号视图
  - `ICPC` 奖项历史
- 排行榜核心指标是 `SCNU Rating`，不是平台原生 `rating`。

## 3. 用户与角色
- 访客：可查看公开用户页与排行榜。
- 用户：可注册、登录、绑定平台账号、查看自己的同步状态。
- 运营：可审核账号归属、维护平台凭证、观察同步链路与告警、人工修正异常数据。

## 4. 账号与用户规则

### 4.1 注册字段
- 用户注册字段固定为：`username`、`email`、`real_name`、`password`。
- `username` 用于登录。
- `real_name` 用于公开展示和 `ICPC` 奖项匹配。

### 4.2 平台账号绑定
- 每个平台允许绑定多个账号。
- 同一个平台账号只能归属一个站内用户。
- 平台账号需要审核通过后才计入统计。
- 用户页和排行榜默认公开可见。

## 5. 过题与展示规则

### 5.1 聚合口径
- 同平台内多个账号按题目唯一标识去重。
- 跨平台不去重。
- 用户主视图只展示 `AC` 列表，不展示完整提交历史。

### 5.2 题目列表字段
- 题号
- 题目名
- 题目链接
- 所属比赛或 `contestId`
- 首次 `AC` 时间

### 5.3 搜索与排序
- 支持按题号搜索和排序。
- 支持按比赛 `ID / contestId` 搜索和排序。
- 搜索和排序都只作用于当前用户、当前视图的数据集。

### 5.4 个人页图表
- 个人页必须展示 `SCNU Rating` 折线图。
- 个人页必须展示类 GitHub contribution 的热力图。
- 热力图统计口径是“每日新 `AC` 题数”。
- `SCNU Rating` 的变化趋势由折线图表达，不靠热力图表达。

## 6. 平台原生分数展示
- `Codeforces` 展示所有已验证账号里的最高 `maxRating`。
- `AtCoder` 只看算法赛最高 `rating`。
- `洛谷` 展示平台可获得的 `rating`。
- 不做“全平台最高分”字段。

## 7. Clist 的定位
- `Clist` 不是“是否 AC”的事实源。
- `Clist` 的主要用途是：
  - 给 `Codeforces / AtCoder` 已通过题补 `problem.rating`
  - 为 `SCNU Rating` 提供题目权重
  - 辅助比赛映射和题目元数据补全
- 在 `Codeforces` 中，`Clist` 是官方 API 的兜底链路。
- 在 `AtCoder` 中，`Clist` 是主链路后的第一层兜底。

## 8. SCNU Rating 规则
- `SCNU Rating` 目前只由 `AtCoder + Codeforces` 两部分组成。
- `洛谷` 和 `ICPC` 不参与 `SCNU Rating`。
- 对每道 `AtCoder / Codeforces` 已通过题，使用对应的 `Clist problem.rating` 套公式计算奖励分。

### 8.1 AtCoder 奖励分公式
设 `Clist rating = r`：

```text
x = 0.001r
p = 8.5x^3 - 16x^2 + 17x + 1
```

- `AT_Best`：近一个月首次通过题目的奖励分前 `20` 和。
- `AT_Recent`：近半个月首次通过题目的奖励分总和。
- `AT_Training = AT_Best * 2.5 + AT_Recent`

### 8.2 Codeforces 奖励分公式
设 `Clist rating = r`：

```text
x = max(0.001r - 0.5, 0)
p = 8.8x^3 - 6x^2 + 10.5x + 1
```

- `CF_Best`：近一个月首次通过题目的奖励分前 `20` 和。
- `CF_Recent`：近半个月首次通过题目的奖励分总和。
- `CF_Training = CF_Best * 2.5 + CF_Recent`

### 8.3 最终得分与排行
- `SCNU Rating = (AT_Training + CF_Training) / 2`
- 排行榜默认按 `SCNU Rating` 排序。
- 后续支持按 `AT_Training / CF_Training / Recent` 排序。
- 需要展示相对昨日 `23:59` 的变化值。
- 需要展示“今日新通过题数”，按每日 `00:00` 清零。

## 9. 平台同步链路

### 9.1 AtCoder
- `AtCoder` 不是单一链路，而是多级兜底。
- 优先级固定为：
  1. 主链路：运营登录 `AtCoder`，运营扩展拿到并长期维护 `Cookie`
  2. 第一层兜底：`Clist`
  3. 第二层兜底：第三方 API
  4. 最后兜底：用户浏览器扩展主动窗口期同步
- `AtCoder` 用户扩展不是主链路，只是最后兜底。
- 用户扩展只上传“刷新窗口内的 `AC` 提交”，不是全部提交。
- 刷新窗口默认是：
  - 上次成功同步时间
  - 到当前触发时间
- 首次同步或窗口缺失时，服务端下发初始化窗口。
- 服务端基于这些窗口内 `AC` 记录归并：
  - 已通过题列表
  - 首次 `AC` 时间
  - 某场比赛 `AC` 了哪些题
- `AtCoder` 任一关键链路失效时需要主动告警：
  - 主链路失效告警
  - `Clist` 失效告警
  - 第三方 API 失效告警

### 9.2 Codeforces
- 主链路：官方 API。
- 兜底：`Clist`。

### 9.3 洛谷
- 只需要一条稳定链路。

### 9.4 ICPC
- 只需要一条稳定链路。
- 只做奖项历史，不进入刷题统计。
- 允许人工修正。

## 10. 页面与功能需求

### 10.1 公开页面
- 用户公开页
- 平台聚合视图
- 单账号视图
- 排行榜

### 10.2 用户公开页
- 展示 `username`、`real_name`。
- 展示 `Codeforces / AtCoder / 洛谷` 聚合过题情况。
- 展示 `ICPC` 奖项历史。
- 展示 `SCNU Rating` 折线图。
- 展示“每日新 `AC` 题数”热力图。

### 10.3 视图切换
- 聚合视图：展示用户所有平台的聚合结果。
- 平台聚合视图：展示指定平台下所有已验证账号去重后的结果。
- 单账号视图：展示某一个平台账号自己的结果。

### 10.4 排行榜
- 默认按 `SCNU Rating` 排序。
- 需要展示：
  - `username`
  - `SCNU Rating`
  - 相对昨日 `23:59` 的变化值
- 详情或展开信息需要展示：
  - `AT_Training`
  - `CF_Training`
  - `Recent`
  - 今日新通过题数

## 11. 技术架构与服务划分

### 11.1 技术栈
- 前端：`React`、`TypeScript`、`Vite`、`React Router`、`shadcn/ui`、`Tailwind CSS`
- 后端：`Go`、`Gin`、`pgxpool`
- 同步抓取：`Python`、`httpx`、`curl_cffi`
- 基础设施：`PostgreSQL`、`Redis`、`NATS + JetStream`、`ElasticSearch`
- 部署：`Docker`、`Docker Compose`

### 11.2 服务划分
- `web`：用户端和管理端统一前端
- `api`：鉴权、用户资料、账号、同步入口、扩展接口
- `scheduler`：周期任务、快照、重试、`Clist` 回填任务投递
- `aggregator`：归并题目事实、计算 `AT / CF / SCNU`、刷新排行榜与索引
- `sync-worker`：访问外部平台与 `Clist`
- `atcoder-extension`：用户本机扩展，同步窗口内 `AC`

## 12. 数据原则
- `PostgreSQL` 是业务真值，也是唯一事实源。
- `Redis` 只做会话、限流、缓存、短期状态。
- `ElasticSearch` 只是派生索引，不是主查询源。
- `Python` 只负责同步，不承载核心业务规则。
- 首 `AC` 事实必须单独维护，后续所有窗口计算都基于 `first_ac_at`。

## 13. 必须长期保持一致的产品口径
- 用户主界面只看 `AC` 列表。
- `AtCoder` 用户扩展只上传窗口内 `AC`。
- `AtCoder` 主链路是运营 `Cookie`，不是用户扩展。
- `SCNU Rating` 只看 `AtCoder + Codeforces`。
- 热力图统计的是“每日新 `AC` 题数”。
- 所有平台尽量有兜底，尤其 `Codeforces / AtCoder`。
