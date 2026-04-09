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
