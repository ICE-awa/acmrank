# 目录迁移后的上下文恢复与后续实现提示

## 1. 当前仓库约定
- 当前工作目录以 `/opt/acmrank` 为准。
- 不再优先相信旧路径 `/mnt/f/archive/acmrank` 中的文件。
- 如果迁移导致上下文缺失，先把理解回填到 `docs/`，再继续编码。

## 2. 恢复上下文的固定顺序
每次接手项目前，先读取：
1. `docs/PRD.md`
2. `docs/technical-architecture.md`
3. `docs/platform-data-integration.md`
4. `docs/atcoder-extension-sync.md`

如果这些文档与“已确认事实”冲突，以“已确认事实”为准，并优先修正文档。

## 3. 必须长期记住的产品口径
- 这是华南师范大学校内使用的竞赛档案与训练排行榜系统。
- 用户页公开，展示 `Codeforces / AtCoder / 洛谷` 聚合过题、平台聚合视图、单账号视图、`ICPC` 奖项历史。
- 用户主视图只展示 `AC` 列表，不展示完整提交历史。
- 个人页必须展示：
  - `SCNU Rating` 折线图
  - 类 GitHub contribution 的热力图
- 热力图统计口径是“每日新 `AC` 题数”。
- `SCNU Rating` 只由 `AtCoder + Codeforces` 组成，不包含 `洛谷` 和 `ICPC`。
- `SCNU Rating` 的题目权重来源是 `Clist problem.rating`。
- `Codeforces` 主链路是官方 API，`Clist` 为兜底。
- `AtCoder` 是多级兜底链路：
  1. 主链路：运营登录 `AtCoder`，运营扩展维护 `Cookie`
  2. 第一层兜底：`Clist`
  3. 第二层兜底：第三方 API
  4. 最后兜底：用户浏览器扩展主动窗口期同步
- `AtCoder` 用户扩展不是主链路。
- `AtCoder` 用户扩展只上传刷新窗口内 `AC` 提交，不上传全部提交。
- 刷新窗口默认是“上次成功同步时间 -> 当前触发时间”。
- 服务端基于这些 `AC` 记录归并：
  - 已通过题列表
  - 首次 `AC` 时间
  - 某场比赛 `AC` 了哪些题
- `AtCoder` 主链路、`Clist`、第三方 API 任一失效时需要主动告警。
- `洛谷` 和 `ICPC` 目前只要求一条稳定链路。
- `PostgreSQL` 是唯一事实源。
- `Redis` 只做会话、限流、缓存、短期状态。
- `NATS + JetStream` 做异步任务。
- `ElasticSearch` 只是派生索引，不是主查询源。

## 4. 后续开发时的硬约束
- 不要假设旧目录文件一定是最新版本。
- 不要在文档与已确认事实冲突时继续编码。
- 不要把 `AtCoder` 用户扩展实现成主同步链路。
- 不要把用户页设计成完整提交历史页面。
- 不要让 `Redis` 或 `ElasticSearch` 承担事实存储。
- 不要把核心排行榜规则写进 Python 同步层。

## 5. 下一阶段实现优先级
1. 保持 `docs/` 与产品口径一致。
2. 保证 `first_ac_at`、平台聚合、比赛 `AC` 汇总的数据模型成立。
3. 保证 `AtCoder` 多级兜底与告警链路成立。
4. 保证 `SCNU Rating`、昨日变化值、今日新 `AC` 数的计算链路成立。
5. 最后再补用户扩展、搜索增强、管理端体验优化。

## 6. 扩展实现提示
- 如果后续需要继续实现 `AtCoder` 扩展，仍然以 `docs/atcoder-extension-sync.md` 为准。
- 扩展的目标是同步窗口内 `AC` 记录，不是抓取完整提交历史。
- 扩展上传结构化结果，不上传原始 Cookie。
