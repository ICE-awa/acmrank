# AtCoder Sync

一个最小可用的 Chrome / Edge Manifest V3 扩展。

它的目标不是拿到原始 AtCoder cookie，而是直接复用用户当前浏览器中的 AtCoder 登录态，在 `atcoder.jp` 页面上下文里抓取：

- 指定用户的比赛历史
- 最近一次提交
- 最近 `N` 场比赛中的去重 `AC` 题号集合

## 目录结构

- `manifest.json`
- `background.js`
- `content-script.js`
- `popup.html`
- `popup.js`

## 为什么要做成浏览器扩展

普通网页前端做不到：

- 读取 `atcoder.jp` 的跨域 cookie
- 读取 `HttpOnly` 的 `REVEL_SESSION`

而浏览器扩展可以：

- 用 `chrome.cookies` 检测是否存在 `REVEL_SESSION`
- 在 `https://atcoder.jp/*` 域上运行 content script
- 让抓取逻辑在 AtCoder 同源上下文里发起请求，减少 `SameSite` 和跨域限制问题

## 已实现能力

1. 检测当前浏览器是否存在 AtCoder 登录态
2. 输入用户名后，读取 `https://atcoder.jp/users/{user}/history/json`
3. 取最近 `N` 场比赛
4. 对每场比赛抓取 `https://atcoder.jp/contests/{contest}/submissions?f.User={user}`
5. 解析：
   - 最新提交的 `contest`
   - `problem_id`
   - `problem_name`
   - `submitted_at`
   - `result`
   - `detail_url`
6. 聚合最近 `N` 场比赛中的去重 `AC` 题号
7. 支持在 popup 中复制：
   - 完整 JSON
   - `solved_problem_ids`

## 加载方式

1. 打开 Chrome 或 Edge 的扩展管理页
2. 开启“开发者模式”
3. 选择“加载已解压的扩展”
4. 选择本目录：`extensions/atcoder-sync/`

## 使用方式

1. 先在同一个浏览器中登录 `https://atcoder.jp/`
2. 打开扩展 popup
3. 确认状态显示“已检测到登录态”
4. 输入 AtCoder 用户名，例如 `treneneno`
5. 设置最近比赛数，默认 `5`
6. 点击“抓取数据”
7. 在 popup 中查看摘要和结构化 JSON

## 返回结果示例

结构化 JSON 中重点字段包括：

```json
{
  "user": "treneneno",
  "latest_submission": {
    "contest": "arc217",
    "problem_id": "arc217_a",
    "problem_name": "A - Min of Sum of XOR",
    "submitted_at": "2026-04-06 00:40:15 +0900",
    "result": "AC",
    "detail_url": "https://atcoder.jp/contests/arc217/submissions/..."
  },
  "solved_problem_ids": ["arc217_a"]
}
```

其中：

- `latest_submission` 是“最近扫描的比赛集合里，时间最新的一条提交”
- `solved_problem_ids` 是“最近 `N` 场比赛的 AC 题号去重结果”

## 权限说明

`manifest.json` 当前使用了这些权限：

- `cookies`
  - 检测 `REVEL_SESSION` 是否存在
- `storage`
  - 保存上次输入的用户名和比赛数
- `tabs`
  - 复用已有的 AtCoder 页签，或临时创建一个后台页签
- `scripting`
  - 在目标页签中确保抓取脚本已注入

`host_permissions`：

- `https://atcoder.jp/*`

## 重要边界

1. 扩展不会把原始 `REVEL_SESSION` 发给任何后端
2. 目前只扫描最近 `N` 场比赛，默认不是全量历史
3. 每场比赛最多扫描 `20` 页提交记录，避免单次请求过重
4. 如果用户没有登录，或登录态失效，提交页抓取会失败
5. `history/json` 能拿到比赛历史，但题目级提交仍依赖登录后提交页解析

## 实现说明

- 登录状态检测在 `background.js`
- 数据抓取和 HTML 解析在 `content-script.js`
- popup 只负责输入、触发和展示结果
- 如果浏览器当前没有打开 AtCoder 页签，后台会临时创建一个非激活页签，抓取结束后自动关闭

## 后续可扩展点

- 增加“扫描更多比赛”与进度展示
- 增加“导出为统一平台数据模型”
- 增加更多字段，例如语言、提交 ID、比赛名、排名
- 增加对整页所有提交的筛选统计
