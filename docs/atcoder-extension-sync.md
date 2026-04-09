# AtCoder 扩展同步设计

## 1. 角色定位
- 本文档只描述 `AtCoder` 用户浏览器扩展这条链路。
- 该链路不是主链路，而是 `AtCoder` 多级兜底中的最后一层。
- 主链路仍然是“运营登录 `AtCoder`，运营扩展长期维护 `Cookie`”。

## 2. 核心规则
- 扩展只上传“刷新窗口内的 `AC` 提交”。
- 扩展不上传全部提交，也不负责构建完整提交历史。
- 刷新窗口默认是“上次成功同步时间 -> 当前触发时间”。
- 首次同步或窗口缺失时，服务端下发初始化窗口。
- 服务端基于这些窗口内 `AC` 记录归并：
  - 已通过题列表
  - 首次 `AC` 时间
  - 某场比赛 `AC` 了哪些题

## 3. 用户流程

### 3.1 首次使用
1. 用户在站内绑定并通过审核的 `AtCoder` 账号。
2. 用户在浏览器中登录 `https://atcoder.jp/`。
3. 用户安装扩展。
4. 用户在站内获取扩展同步令牌。
5. 扩展完成绑定后，用户触发同步。

### 3.2 日常使用
1. 扩展检测当前浏览器中的 `AtCoder` 登录态。
2. 扩展向服务端请求本次同步窗口。
3. 扩展发现本次需要扫描的比赛集合。
4. 扩展仅提取窗口内的 `AC` 记录。
5. 扩展分批上传 `AC` 记录。
6. 服务端归并并更新 `last_success_synced_at`。

## 4. 鉴权与安全
- 推荐使用站内短期 `sync_token`。
- 扩展不能上传原始 `REVEL_SESSION` 或其他原始 Cookie。
- 扩展只上传结构化 `AC` 记录。
- `sync_token` 必须支持吊销、轮换和过期。

## 5. 刷新窗口

### 5.1 默认窗口
- `window_from = 上次成功同步时间`
- `window_to = 当前触发时间`

### 5.2 初始化窗口
- 首次同步或窗口缺失时，由服务端下发初始化窗口。
- 初始化窗口大小由服务端配置，不由扩展自行决定。

### 5.3 服务端状态
服务端至少记录：
- `site_user_id`
- `atcoder_handle`
- `last_success_synced_at`
- `last_attempt_synced_at`
- `last_extension_sync_status`

## 6. 比赛发现策略
- 扩展需要先确定“本次应扫描哪些比赛”。
- 建议候选集合取并集：
  - 官方 `history/json` 中可见的比赛
  - 服务端已知的该账号历史比赛
- 这是最后兜底链路，不要求替代主链路做无限历史发现。

## 7. 抓取与过滤规则

### 7.1 抓取入口
- 在用户已登录 `AtCoder` 的浏览器环境中抓取官方页面。
- 推荐按比赛抓取用户提交页。

### 7.2 过滤口径
- 只保留满足 `window_from <= accepted_at <= window_to` 的 `AC` 记录。
- 非 `AC` 记录不上传。

### 7.3 去重键
推荐优先级：
1. `submission_id`
2. `detail_url`
3. `handle + task_id + accepted_at`

## 8. 上传结构
每条记录建议至少包含：

```json
{
  "platform": "atcoder",
  "handle": "treneneno",
  "contest_id": "abc451",
  "contest_name": "AtCoder Beginner Contest 451",
  "task_id": "abc451_f",
  "task_short": "F",
  "task_name": "Make Bipartite 3",
  "accepted_at": "2026-04-08T20:57:30+09:00",
  "submission_id": "74734781",
  "detail_url": "https://atcoder.jp/contests/abc451/submissions/74734781",
  "source": "atcoder_extension",
  "fetched_at": "2026-04-09T02:00:00+08:00"
}
```

## 9. 接口建议

### 9.1 初始化同步
`POST /api/extension/atcoder/init`

请求：

```json
{
  "sync_token": "xxx"
}
```

返回：

```json
{
  "site_user_id": 123,
  "server_time": "2026-04-09T02:00:00+08:00",
  "handles": ["treneneno"],
  "window_from": "2026-04-01T00:00:00+08:00",
  "window_to": "2026-04-09T02:00:00+08:00",
  "known_contests": ["abc451", "abc452"],
  "max_pages_per_contest": 20,
  "batch_size": 200
}
```

### 9.2 上传 AC 批次
`POST /api/extension/atcoder/upload`

请求：

```json
{
  "sync_token": "xxx",
  "handle": "treneneno",
  "window_from": "2026-04-01T00:00:00+08:00",
  "window_to": "2026-04-09T02:00:00+08:00",
  "ac_records": []
}
```

返回：

```json
{
  "accepted": 128,
  "deduplicated": 12,
  "next_cursor": null
}
```

### 9.3 完成同步
`POST /api/extension/atcoder/complete`

请求：

```json
{
  "sync_token": "xxx",
  "handle": "treneneno",
  "window_from": "2026-04-01T00:00:00+08:00",
  "window_to": "2026-04-09T02:00:00+08:00",
  "uploaded_count": 128,
  "status": "success"
}
```

返回：

```json
{
  "status": "ok",
  "last_success_synced_at": "2026-04-09T02:00:00+08:00"
}
```

## 10. 服务端归并规则

### 10.1 原始事件表
建议先落 `atcoder_ac_event_raw`：
- `submission_id`
- `handle`
- `contest_id`
- `task_id`
- `accepted_at`
- `detail_url`
- `raw_payload`

### 10.2 题目事实表
归并出 `atcoder_problem_fact`：
- `handle`
- `task_id`
- `first_ac_at`
- `first_ac_source`
- `first_ac_submission_id`
- `latest_ac_at`

### 10.3 比赛 AC 汇总
归并出 `contest_ac_summary`：
- `handle`
- `contest_id`
- `ac_task_ids`
- `ac_count`

## 11. 与 Clist 的关系
- 扩展上传后，服务端先确认 `AC` 事实。
- 后续单独跑 `Clist` 补全任务，为这些已通过题补：
  - `clist_problem_id`
  - `clist_rating`
- `Clist` 仍然不是该链路中的 `AC` 事实源。

## 12. 失败处理
- 上传接口按 `submission_id` 幂等去重。
- 完成同步前，不推进 `last_success_synced_at`。
- 扩展侧可对网络失败做有限重试。
- 扩展失败不会改变“主链路不是用户扩展”的产品定义。

## 13. 验收标准
满足以下条件即可视为该链路可用：
1. 用户能完成扩展绑定。
2. 扩展能检测 `AtCoder` 登录态。
3. 扩展能拿到服务端下发的刷新窗口。
4. 扩展能抓到窗口内 `AC` 记录。
5. 服务端能保存并去重这些 `AC` 记录。
6. 用户页能基于这些记录展示：
   - `AC` 题列表
   - 首次 `AC` 时间
   - 某场比赛 `AC` 了哪些题
7. 这些题后续可以继续补 `Clist rating`。
