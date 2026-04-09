# ACMRank 平台数据接入文档

## 1. 文档约定
- 本文档用于定义 `Codeforces / AtCoder / 洛谷 / ICPC / Clist` 的接入边界。
- 在目录迁移后，先读取：
  - `docs/PRD.md`
  - `docs/technical-architecture.md`
  - `docs/platform-data-integration.md`
  - `docs/atcoder-extension-sync.md`
- 如果接入实现与“已确认事实”冲突，先修正文档，再继续开发。

## 2. 总体接入原则
- `PostgreSQL` 是唯一事实源。
- 平台接入层负责拉取、结构化、标记来源、保存原始响应引用。
- `Clist` 不是“是否 AC”的事实源。
- 用户主界面只展示 `AC` 列表，不展示完整提交历史。
- 跨平台不去重，同平台内按题目唯一标识去重。
- `first_ac_at` 必须单独维护，不能每次临时从原始记录扫描得出。

## 3. 统一输出模型

### 3.1 账号资料
- `platform`
- `handle`
- `display_name`
- `rating`
- `max_rating`
- `profile_url`
- `source`
- `fetched_at`

### 3.2 AC 记录
- `platform`
- `handle`
- `problem_key`
- `contest_id`
- `problem_index_or_task_id`
- `problem_name`
- `problem_url`
- `accepted_at`
- `submission_id_or_ref`
- `source`
- `source_url`
- `raw_payload_ref`

### 3.3 比赛与奖项
- 比赛记录：
  - `contest_id`
  - `contest_name`
  - `rank`
  - `rating_delta`
  - `ended_at`
- 奖项记录：
  - `contest_name`
  - `award_name`
  - `rank_text`
  - `date`
  - `source_url`

## 4. Clist 的角色
- 为 `Codeforces / AtCoder` 已通过题补 `problem.rating`。
- 这些 `rating` 用于计算 `SCNU Rating`。
- 可辅助比赛映射、题目元数据补全、资源对齐。
- 在 `Codeforces` 中是官方 API 的兜底链路。
- 在 `AtCoder` 中是主链路后的第一层兜底。
- 即使走兜底链路，也必须保留 `source` 与 `source_url`，便于后续回补和纠正。

## 5. Codeforces

### 5.1 接入目标
- 获取账号资料。
- 获取 `AC` 题目集合与首次 `AC` 时间。
- 获取比赛历史。
- 为已通过题补 `Clist rating`。
- 为公开页展示最高 `maxRating`。

### 5.2 链路优先级
1. 主链路：官方 API
2. 兜底：`Clist`

### 5.3 主链路建议
- `user.info`
  - 资料与 `rating / maxRating`
- `user.status`
  - `verdict = OK` 的提交
- `user.rating`
  - 比赛历史

### 5.4 规范化规则
- `problem_key = CF-{contestId}{index}`
- `accepted_at` 取首次 `OK` 的提交时间
- 平台展示分数取所有已验证账号中的最高 `maxRating`
- 同一用户同平台多账号按 `problem_key` 去重

### 5.5 Clist 用法
- 回填 `problem.rating`
- 补比赛与题目元数据
- 当官方 API 异常时作为兜底链路，但需要保留来源并等待官方链路恢复后复核

## 6. AtCoder

### 6.1 产品结论
- `AtCoder` 不是单一同步通路，而是多级兜底。
- 用户扩展不是主链路，只是最后兜底。
- 用户扩展只上传窗口内 `AC` 提交，不上传全部提交。

### 6.2 固定链路顺序
1. 主链路：运营登录 `AtCoder`，由运营扩展获取并长期维护 `Cookie`
2. 第一层兜底：`Clist`
3. 第二层兜底：第三方 API
4. 最后兜底：用户浏览器扩展主动窗口期同步

### 6.3 主链路职责
- 基于运营 `Cookie` 访问官方页面或官方域名可用接口。
- 负责拿到尽可能完整的：
  - 账号资料
  - 比赛历史
  - `AC` 题目事实
  - `first_ac_at`
  - 某场比赛 `AC` 了哪些题
- 这是默认生产链路，扩展不应替代它。

### 6.4 Clist 兜底职责
- 回填 `problem.rating`
- 补比赛映射与题目元数据
- 在主链路不稳定时，辅助维持题目资源映射和统计链路连续性
- 仍然不能把 `Clist` 定义成“是否 AC”的最终事实源

### 6.5 第三方 API 兜底职责
- 作为主链路和 `Clist` 之后的第二层兜底
- 用于恢复部分比赛、题目或近期 `AC` 候选数据
- 结果必须保留来源，并允许后续被更高可信来源纠正

### 6.6 用户扩展兜底职责
- 只在前 3 层链路不足时承担用户侧补数作用
- 上传内容固定为“窗口内 `AC` 记录”
- 刷新窗口默认是“上次成功同步时间 -> 当前触发时间”
- 首次同步或窗口缺失时，服务端下发初始化窗口
- 服务端根据这些记录归并：
  - 已通过题列表
  - `first_ac_at`
  - 某场比赛 `AC` 了哪些题

### 6.7 平台展示规则
- 平台展示分数只看算法赛最高 `rating`
- 同平台多账号按题目唯一标识去重

### 6.8 AtCoder 告警要求
- 主链路失效告警
- `Clist` 失效告警
- 第三方 API 失效告警

## 7. 洛谷

### 7.1 接入目标
- 获取账号资料
- 获取去重后的过题列表
- 获取平台 `rating`

### 7.2 约束
- 只要求一条稳定链路
- 不参与 `SCNU Rating`
- 允许通过稳定的官方页面接口或解析链路实现

## 8. ICPC

### 8.1 接入目标
- 只同步奖项历史
- 以真实姓名匹配为主

### 8.2 约束
- 不进入刷题统计
- 允许人工修正
- 只要求一条稳定链路

## 9. 归并规则

### 9.1 平台内去重
- `Codeforces`：按规范化 `problem_key` 去重
- `AtCoder`：按 `task_id` 或统一任务键去重
- `洛谷`：按平台题号去重

### 9.2 跨平台
- 不去重，直接累加展示

### 9.3 first_ac_at
- 对每个平台题目独立维护 `first_ac_at`
- 后续所有统计窗口、热力图、`AT_Best / AT_Recent / CF_Best / CF_Recent` 都基于该字段

## 10. SCNU Rating 依赖链
- 只消费 `AtCoder + Codeforces` 的去重 `AC` 事实
- 只消费这两个平台上已回填的 `Clist problem.rating`
- `洛谷` 与 `ICPC` 不进入 `SCNU Rating`
- `aggregator` 负责统一计算，不在同步层分散实现

## 11. 接入落地要求
- 所有同步记录必须包含 `source`。
- 所有非平凡回填都要能追溯到 `source_url` 或 `raw_payload_ref`。
- 任一平台账号只能归属一个站内用户。
- 未审核通过的账号不能进入聚合与排行榜统计。
- 接入层只负责同步，不负责定义用户公开页展示逻辑。
