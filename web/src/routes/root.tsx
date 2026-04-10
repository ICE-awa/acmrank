import { useState, type CSSProperties, type ReactNode } from "react";

type PreviewId = "portal" | "profile" | "ranking";

type PreviewOption = {
  id: PreviewId;
  label: string;
  eyebrow: string;
  mood: string;
  recommendation: string;
  accent: string;
  palette: string[];
};

type Tokens = {
  page: string;
  panel: string;
  panelAlt: string;
  line: string;
  text: string;
  textMuted: string;
  textSoft: string;
  accent: string;
  accentSoft: string;
  chart: string;
  heatmap: [string, string, string, string, string];
};

type MetricCard = {
  label: string;
  value: string;
  detail: string;
  highlight?: boolean;
};

type ProblemRow = {
  id: string;
  contest: string;
  rating: string;
  time: string;
};

const previewOptions: PreviewOption[] = [
  {
    id: "portal",
    label: "门户布局",
    eyebrow: "首页入口优先",
    mood: "公告、榜单、个人入口在同一页，节奏更活。",
    recommendation: "适合作为默认落地页，先看全站动态再下钻。",
    accent: "#3498db",
    palette: ["#34495e", "#f3f5f7", "#ffffff", "#e74c3c", "#2ea043"],
  },
  {
    id: "profile",
    label: "个人布局",
    eyebrow: "公开主页优先",
    mood: "把个人资料、曲线、热力图作为第一视觉层。",
    recommendation: "适合先完成公开个人页这条主链路。",
    accent: "#3498db",
    palette: ["#34495e", "#f6f8fa", "#ffffff", "#f39c12", "#2ea043"],
  },
  {
    id: "ranking",
    label: "榜单布局",
    eyebrow: "排行榜优先",
    mood: "榜单表格和走势占主屏，信息关系最直接。",
    recommendation: "适合作为站点核心页面，突出竞争和变化值。",
    accent: "#3498db",
    palette: ["#34495e", "#f3f5f7", "#ffffff", "#9b59b6", "#2ea043"],
  },
];

const luoguTokens: Tokens = {
  page: "#f3f5f7",
  panel: "#ffffff",
  panelAlt: "#f8fbff",
  line: "#d8e1e8",
  text: "#34495e",
  textMuted: "#5c6b78",
  textSoft: "#8191a0",
  accent: "#3498db",
  accentSoft: "rgba(52, 152, 219, 0.12)",
  chart: "#3498db",
  heatmap: ["#ffffff", "#ebf7e8", "#cfe9d1", "#78c27d", "#2ea043"],
};

const modulePills = [
  { label: "Problems", color: "#e74c3c" },
  { label: "Training", color: "#f39c12" },
  { label: "Contests", color: "#9b59b6" },
  { label: "Teams", color: "#3498db" },
  { label: "Discuss", color: "#16a085" },
] as const;

const metricCards: MetricCard[] = [
  {
    label: "SCNU Rating",
    value: "2476",
    detail: "较昨日 +46",
    highlight: true,
  },
  { label: "今日新 AC", value: "12", detail: "每日 00:00 清零" },
  { label: "已验证账号", value: "7", detail: "CF / AT / 洛谷" },
];

const leaderboardRows = [
  { rank: "01", user: "treneneno", score: "2476", delta: "+46" },
  { rank: "02", user: "ice", score: "2431", delta: "+19" },
  { rank: "03", user: "sherry", score: "2388", delta: "+11" },
  { rank: "04", user: "frost", score: "2340", delta: "-6" },
  { rank: "05", user: "lina", score: "2302", delta: "+3" },
] as const;

const problemRows: ProblemRow[] = [
  {
    id: "CF-2048C",
    contest: "Codeforces 2048",
    rating: "2100",
    time: "2026-04-09 22:14",
  },
  {
    id: "abc451_f",
    contest: "AtCoder ABC451",
    rating: "1900",
    time: "2026-04-08 20:57",
  },
  { id: "P3373", contest: "洛谷", rating: "LG", time: "2026-04-08 18:03" },
  {
    id: "CF-2039D",
    contest: "Codeforces 2039",
    rating: "1700",
    time: "2026-04-07 23:11",
  },
];

const awardRows = [
  "ICPC EC Final 2025 银奖",
  "广东省赛 2025 金奖",
  "校队选拔训练营 2026 A 组",
] as const;

const platformRows = [
  { label: "Codeforces", solved: 634, ratio: 0.52, color: "#3498db" },
  { label: "AtCoder", solved: 428, ratio: 0.35, color: "#f39c12" },
  { label: "洛谷", solved: 154, ratio: 0.13, color: "#16a085" },
] as const;

const heatmapWeeks = [
  [0, 1, 0, 2, 0, 1, 3],
  [1, 0, 2, 3, 1, 0, 0],
  [2, 3, 4, 1, 0, 1, 2],
  [0, 0, 1, 2, 3, 4, 2],
  [1, 2, 0, 0, 2, 3, 1],
  [3, 4, 2, 1, 0, 1, 0],
  [2, 1, 3, 2, 4, 2, 1],
  [0, 1, 1, 2, 0, 3, 4],
  [1, 0, 2, 4, 3, 2, 0],
  [2, 3, 1, 0, 1, 4, 2],
] as const;

const ratingTrend = [
  1920, 1968, 2012, 2054, 2096, 2140, 2212, 2280, 2336, 2398, 2430, 2476,
] as const;

const fontStacks = {
  sans: '"IBM Plex Sans", "Noto Sans SC", "PingFang SC", sans-serif',
  mono: '"IBM Plex Mono", "JetBrains Mono", monospace',
} as const;

function buildChartGeometry(
  values: readonly number[],
  width: number,
  height: number,
) {
  const padding = 16;
  const min = Math.min(...values);
  const max = Math.max(...values);
  const range = max - min || 1;
  const step = (width - padding * 2) / Math.max(values.length - 1, 1);

  const points = values.map((value, index) => {
    const x = padding + step * index;
    const y =
      height -
      padding -
      ((value - min) / range) * Math.max(height - padding * 2, 1);
    return { x, y };
  });

  return {
    points,
    line: points.map(({ x, y }) => `${x},${y}`).join(" "),
    area: [
      `M ${points[0]?.x ?? padding} ${height - padding}`,
      ...points.map(({ x, y }) => `L ${x} ${y}`),
      `L ${points.at(-1)?.x ?? padding} ${height - padding}`,
      "Z",
    ].join(" "),
  };
}

function Panel({
  children,
  className,
  style,
}: {
  children: ReactNode;
  className?: string;
  style?: CSSProperties;
}) {
  return (
    <article
      className={`rounded-2xl border p-4 sm:p-5 ${className ?? ""}`}
      style={style}
    >
      {children}
    </article>
  );
}

function Stage({ children, tokens }: { children: ReactNode; tokens: Tokens }) {
  return (
    <section
      className="overflow-hidden rounded-[28px] border shadow-[0_24px_56px_rgba(0,0,0,0.12)]"
      style={{
        background: tokens.page,
        borderColor: tokens.line,
        color: tokens.text,
        fontFamily: fontStacks.sans,
      }}
    >
      {children}
    </section>
  );
}

function TopBar({ tokens }: { tokens: Tokens }) {
  return (
    <header
      className="border-b px-5 py-4 sm:px-6"
      style={{ borderColor: tokens.line, background: tokens.panel }}
    >
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <span
            className="inline-flex h-8 min-w-8 items-center justify-center rounded-lg px-2 text-sm font-semibold text-white"
            style={{ background: tokens.accent }}
          >
            A
          </span>
          <div>
            <p className="text-sm font-semibold">ACMRank</p>
            <p className="text-xs" style={{ color: tokens.textSoft }}>
              Luogu 色感 + 中性色骨架
            </p>
          </div>
        </div>
        <div className="flex flex-wrap gap-2 text-xs">
          {modulePills.map((pill) => (
            <span
              key={pill.label}
              className="rounded-full px-3 py-1 text-white"
              style={{ background: pill.color }}
            >
              {pill.label}
            </span>
          ))}
        </div>
      </div>
    </header>
  );
}

function PreviewSidebar({
  title,
  items,
  tokens,
}: {
  title: string;
  items: string[];
  tokens: Tokens;
}) {
  return (
    <aside
      className="min-h-full border-r py-4"
      style={{ background: tokens.panel, borderColor: tokens.line }}
    >
      <p
        className="px-5 text-[11px] uppercase tracking-[0.28em]"
        style={{ color: tokens.textSoft }}
      >
        导航
      </p>
      <h3 className="mt-3 px-5 text-lg font-semibold">{title}</h3>
      <nav className="mt-3 space-y-1">
        {items.map((item, index) => (
          <a
            key={item}
            href="#"
            className="flex items-center justify-between border-l-2 px-5 py-2.5 text-sm no-underline transition-[background-color,border-color] duration-200"
            style={{
              borderLeftColor: index === 0 ? tokens.accent : "transparent",
              background: index === 0 ? tokens.accentSoft : tokens.panelAlt,
              color: index === 0 ? tokens.text : tokens.textMuted,
            }}
          >
            <span>{item}</span>
            <span
              style={{ color: index === 0 ? tokens.accent : tokens.textSoft }}
            >
              {index === 0 ? "●" : "○"}
            </span>
          </a>
        ))}
      </nav>
    </aside>
  );
}

function SectionHead({
  title,
  detail,
  tokens,
}: {
  title: string;
  detail?: string;
  tokens: Tokens;
}) {
  return (
    <div className="mb-4 flex items-end justify-between gap-3">
      <h3 className="text-xl font-semibold sm:text-2xl">{title}</h3>
      {detail ? (
        <span className="text-sm" style={{ color: tokens.textSoft }}>
          {detail}
        </span>
      ) : null}
    </div>
  );
}

function MetricGrid({
  cards,
  tokens,
}: {
  cards: MetricCard[];
  tokens: Tokens;
}) {
  return (
    <div className="grid gap-3 sm:grid-cols-3">
      {cards.map((card) => (
        <div
          key={card.label}
          className="rounded-xl border px-4 py-3"
          style={{ background: tokens.panelAlt, borderColor: tokens.line }}
        >
          <p
            className="text-[11px] uppercase tracking-[0.2em]"
            style={{ color: card.highlight ? tokens.accent : tokens.textMuted }}
          >
            {card.label}
          </p>
          <p className="mt-2 text-2xl font-semibold">{card.value}</p>
          <p className="mt-1 text-sm" style={{ color: tokens.textSoft }}>
            {card.detail}
          </p>
        </div>
      ))}
    </div>
  );
}

function MetricStrip({
  cards,
  tokens,
}: {
  cards: MetricCard[];
  tokens: Tokens;
}) {
  return (
    <div className="grid gap-2 sm:grid-cols-3">
      {cards.map((card, index) => (
        <div
          key={card.label}
          className="px-1 py-1 sm:px-3"
          style={{
            borderLeft: index === 0 ? "none" : `1px solid ${tokens.line}`,
          }}
        >
          <p
            className="text-[11px] uppercase tracking-[0.2em]"
            style={{ color: card.highlight ? tokens.accent : tokens.textMuted }}
          >
            {card.label}
          </p>
          <p className="mt-1.5 text-2xl font-semibold">{card.value}</p>
          <p className="mt-1 text-sm" style={{ color: tokens.textSoft }}>
            {card.detail}
          </p>
        </div>
      ))}
    </div>
  );
}

function RankingList({ tokens }: { tokens: Tokens }) {
  return (
    <div className="space-y-2.5">
      {leaderboardRows.map((row) => (
        <div
          key={row.rank}
          className="grid grid-cols-[42px_minmax(0,1fr)_74px_56px] items-center gap-3 rounded-xl border px-3 py-2.5 text-sm"
          style={{ background: tokens.panelAlt, borderColor: tokens.line }}
        >
          <span
            style={{ color: tokens.textMuted, fontFamily: fontStacks.mono }}
          >
            {row.rank}
          </span>
          <span className="truncate">{row.user}</span>
          <span className="text-right">{row.score}</span>
          <span
            className="text-right"
            style={{ color: row.delta.startsWith("+") ? "#2ea043" : "#d94848" }}
          >
            {row.delta}
          </span>
        </div>
      ))}
    </div>
  );
}

function ProblemTable({
  tokens,
  rows = problemRows,
}: {
  tokens: Tokens;
  rows?: ProblemRow[];
}) {
  return (
    <div className="space-y-2.5">
      {rows.map((problem) => (
        <div
          key={`${problem.id}-${problem.time}`}
          className="grid gap-2 rounded-xl border px-3 py-3 sm:grid-cols-[1fr_80px_150px]"
          style={{ background: tokens.panelAlt, borderColor: tokens.line }}
        >
          <div>
            <p className="text-sm font-medium">{problem.id}</p>
            <p className="mt-1 text-xs" style={{ color: tokens.textSoft }}>
              {problem.contest}
            </p>
          </div>
          <p className="text-sm sm:text-right" style={{ color: tokens.accent }}>
            {problem.rating}
          </p>
          <p
            className="text-sm sm:text-right"
            style={{ color: tokens.textSoft }}
          >
            {problem.time}
          </p>
        </div>
      ))}
    </div>
  );
}

function Awards({ tokens }: { tokens: Tokens }) {
  return (
    <div className="space-y-2.5">
      {awardRows.map((award) => (
        <div
          key={award}
          className="rounded-xl border px-3 py-2.5 text-sm"
          style={{ background: tokens.panelAlt, borderColor: tokens.line }}
        >
          {award}
        </div>
      ))}
    </div>
  );
}

function Heatmap({ tokens }: { tokens: Tokens }) {
  return (
    <div className="grid grid-flow-col grid-rows-7 gap-1.5 overflow-x-auto py-0.5">
      {heatmapWeeks.flatMap((week, weekIndex) =>
        week.map((value, dayIndex) => (
          <div
            key={`${weekIndex}-${dayIndex}`}
            className="h-4 w-4 rounded-[4px] border"
            style={{
              background: tokens.heatmap[value],
              borderColor: tokens.line,
            }}
            title={`Week ${weekIndex + 1}, day ${dayIndex + 1}: ${value} new AC`}
          />
        )),
      )}
    </div>
  );
}

function TrendChart({ tokens }: { tokens: Tokens }) {
  const { points, line, area } = buildChartGeometry(ratingTrend, 420, 216);

  return (
    <svg viewBox="0 0 420 216" className="h-52 w-full">
      {[54, 98, 142].map((y) => (
        <line
          key={y}
          x1="16"
          x2="404"
          y1={y}
          y2={y}
          stroke={tokens.line}
          strokeDasharray="6 8"
        />
      ))}
      <path d={area} fill={tokens.accentSoft} />
      <polyline
        fill="none"
        points={line}
        stroke={tokens.chart}
        strokeWidth="3"
        strokeLinecap="round"
      />
      {points.map(({ x, y }) => (
        <circle key={`${x}-${y}`} cx={x} cy={y} r="3.5" fill={tokens.chart} />
      ))}
    </svg>
  );
}

function PlatformBreakdown({ tokens }: { tokens: Tokens }) {
  return (
    <div className="space-y-3">
      {platformRows.map((row) => (
        <div key={row.label} className="space-y-1.5">
          <div className="flex items-center justify-between text-sm">
            <span>{row.label}</span>
            <span style={{ color: tokens.textMuted }}>{row.solved}</span>
          </div>
          <div
            className="h-2 overflow-hidden rounded-full"
            style={{ background: "#e8edf2" }}
          >
            <div
              className="h-full rounded-full"
              style={{ width: `${row.ratio * 100}%`, background: row.color }}
            />
          </div>
        </div>
      ))}
    </div>
  );
}

function DashboardLayout({
  heading,
  description,
  sidebarTitle,
  sidebarItems,
  children,
}: {
  heading: string;
  description: string;
  sidebarTitle: string;
  sidebarItems: string[];
  children: ReactNode;
}) {
  const tokens = luoguTokens;

  return (
    <Stage tokens={tokens}>
      <TopBar tokens={tokens} />

      <div className="grid xl:grid-cols-[228px_minmax(0,1fr)]">
        <PreviewSidebar
          title={sidebarTitle}
          items={sidebarItems}
          tokens={tokens}
        />

        <div className="space-y-4 p-4 sm:p-5">
          <Panel
            style={{ background: tokens.panel, borderColor: tokens.line }}
            className="px-5 py-4"
          >
            <h2 className="text-2xl font-semibold sm:text-3xl">{heading}</h2>
            <p
              className="mt-2 text-sm leading-7"
              style={{ color: tokens.textMuted }}
            >
              {description}
            </p>
          </Panel>
          {children}
        </div>
      </div>
    </Stage>
  );
}

function PortalPreview() {
  const tokens = luoguTokens;

  return (
    <DashboardLayout
      heading="门户布局：总览 + 公告 + 榜单"
      description="保留 Luogu 风格的彩色入口，但主界面用黑白灰做骨架，首页先呈现最关键的动态。"
      sidebarTitle="首页导航"
      sidebarItems={["总览", "排行榜", "个人页", "训练", "公告", "讨论"]}
    >
      <div className="grid gap-4 xl:grid-cols-[1.05fr_0.95fr]">
        <Panel style={{ background: tokens.panel, borderColor: tokens.line }}>
          <SectionHead title="核心指标" detail="每日更新" tokens={tokens} />
          <MetricGrid cards={metricCards} tokens={tokens} />
        </Panel>

        <Panel style={{ background: tokens.panel, borderColor: tokens.line }}>
          <SectionHead title="站内动态" detail="最新公告" tokens={tokens} />
          <div
            className="space-y-2.5 text-sm"
            style={{ color: tokens.textMuted }}
          >
            <p>账号绑定审核每周二、周五统一处理。</p>
            <p>排行榜每天 23:59 生成快照，次日展示变化值。</p>
            <p>AtCoder 主链路失败会触发主动告警。</p>
          </div>
        </Panel>
      </div>

      <div className="grid gap-4 xl:grid-cols-[1.05fr_0.95fr]">
        <Panel style={{ background: tokens.panel, borderColor: tokens.line }}>
          <SectionHead title="排行榜" detail="SCNU Rating" tokens={tokens} />
          <RankingList tokens={tokens} />
        </Panel>

        <div className="space-y-4">
          <Panel style={{ background: tokens.panel, borderColor: tokens.line }}>
            <SectionHead
              title="Daily New AC"
              detail="GitHub 风格绿色热力图"
              tokens={tokens}
            />
            <Heatmap tokens={tokens} />
          </Panel>

          <Panel style={{ background: tokens.panel, borderColor: tokens.line }}>
            <SectionHead title="近期过题" detail="最新 AC" tokens={tokens} />
            <ProblemTable tokens={tokens} rows={problemRows.slice(0, 3)} />
          </Panel>
        </div>
      </div>
    </DashboardLayout>
  );
}

function ProfilePreview() {
  const tokens = luoguTokens;

  return (
    <DashboardLayout
      heading="个人布局：信息卡 + 曲线 + 过题"
      description="这一版把用户个人信息放到第一屏，图表和过题列表在同一阅读流里，适合公开个人页。"
      sidebarTitle="个人页导航"
      sidebarItems={[
        "个人概览",
        "SCNU Rating",
        "Daily New AC",
        "过题列表",
        "奖项历史",
      ]}
    >
      <Panel style={{ background: tokens.panel, borderColor: tokens.line }}>
        <div className="grid gap-4 lg:grid-cols-[260px_minmax(0,1fr)]">
          <div
            className="rounded-xl border p-4"
            style={{ background: tokens.panelAlt, borderColor: tokens.line }}
          >
            <p className="text-sm" style={{ color: tokens.accent }}>
              treneneno
            </p>
            <h3 className="mt-1 text-xl font-semibold">公开个人页</h3>
            <div
              className="mt-3 space-y-1.5 text-sm"
              style={{ color: tokens.textMuted }}
            >
              <p>实名：陈某某</p>
              <p>已验证账号：7</p>
              <p>ICPC 奖项：3</p>
            </div>
          </div>

          <MetricGrid cards={metricCards} tokens={tokens} />
        </div>
      </Panel>

      <div className="grid gap-4 xl:grid-cols-[1.08fr_0.92fr]">
        <Panel style={{ background: tokens.panel, borderColor: tokens.line }}>
          <SectionHead
            title="SCNU Rating"
            detail="近 12 个时间点"
            tokens={tokens}
          />
          <TrendChart tokens={tokens} />
        </Panel>
        <Panel style={{ background: tokens.panel, borderColor: tokens.line }}>
          <SectionHead
            title="Daily New AC"
            detail="绿色热力图"
            tokens={tokens}
          />
          <Heatmap tokens={tokens} />
        </Panel>
      </div>

      <div className="grid gap-4 xl:grid-cols-[0.95fr_1.05fr]">
        <Panel style={{ background: tokens.panel, borderColor: tokens.line }}>
          <SectionHead title="ICPC 奖项历史" tokens={tokens} />
          <Awards tokens={tokens} />
        </Panel>

        <Panel style={{ background: tokens.panel, borderColor: tokens.line }}>
          <SectionHead title="过题列表" tokens={tokens} />
          <ProblemTable tokens={tokens} />
        </Panel>
      </div>
    </DashboardLayout>
  );
}

function RankingPreview() {
  const tokens = luoguTokens;

  return (
    <DashboardLayout
      heading="榜单布局：排行榜优先"
      description="核心信息先给榜单和变化值，曲线和热力图放在右侧辅助区，整体更接近成熟竞赛站 Dashboard。"
      sidebarTitle="榜单导航"
      sidebarItems={[
        "总榜",
        "SCNU Rating",
        "昨日变化",
        "Daily New AC",
        "平台拆分",
      ]}
    >
      <section
        className="overflow-hidden rounded-2xl border"
        style={{ background: tokens.panel, borderColor: tokens.line }}
      >
        <div className="px-5 py-4">
          <p
            className="text-sm font-medium"
            style={{ color: tokens.textMuted }}
          >
            今日快照
          </p>
          <div className="mt-3">
            <MetricStrip cards={metricCards} tokens={tokens} />
          </div>
        </div>

        <div
          className="border-t px-5 py-4"
          style={{ borderColor: tokens.line }}
        >
          <div className="grid gap-5 xl:grid-cols-[1.06fr_0.94fr]">
            <div>
              <p
                className="mb-3 text-sm font-medium"
                style={{ color: tokens.textMuted }}
              >
                排行榜（SCNU Rating）
              </p>
              <RankingList tokens={tokens} />
            </div>

            <div className="space-y-5">
              <div>
                <p
                  className="mb-3 text-sm font-medium"
                  style={{ color: tokens.textMuted }}
                >
                  评分走势
                </p>
                <TrendChart tokens={tokens} />
              </div>
              <div>
                <p
                  className="mb-3 text-sm font-medium"
                  style={{ color: tokens.textMuted }}
                >
                  Daily New AC（绿色热力图）
                </p>
                <Heatmap tokens={tokens} />
              </div>
            </div>
          </div>
        </div>

        <div
          className="border-t px-5 py-4"
          style={{ borderColor: tokens.line }}
        >
          <div className="grid gap-5 xl:grid-cols-[1fr_1fr]">
            <div>
              <p
                className="mb-3 text-sm font-medium"
                style={{ color: tokens.textMuted }}
              >
                近期过题
              </p>
              <ProblemTable tokens={tokens} rows={problemRows.slice(0, 3)} />
            </div>
            <div>
              <p
                className="mb-3 text-sm font-medium"
                style={{ color: tokens.textMuted }}
              >
                平台拆分
              </p>
              <PlatformBreakdown tokens={tokens} />
            </div>
          </div>
        </div>
      </section>
    </DashboardLayout>
  );
}

function PreviewButton({
  option,
  active,
  onSelect,
}: {
  option: PreviewOption;
  active: boolean;
  onSelect: (id: PreviewId) => void;
}) {
  return (
    <button
      type="button"
      onClick={() => onSelect(option.id)}
      aria-pressed={active}
      className="w-full rounded-xl border px-4 py-3 text-left transition-[border-color,background-color] duration-200"
      style={{
        borderColor: active ? option.accent : "#d7dee6",
        background: active ? "rgba(52, 152, 219, 0.08)" : "#ffffff",
      }}
    >
      <p
        className="text-[11px] uppercase tracking-[0.24em]"
        style={{ color: "#8191a0" }}
      >
        {option.eyebrow}
      </p>
      <h2 className="mt-2 text-lg font-semibold" style={{ color: "#34495e" }}>
        {option.label}
      </h2>
      <p className="mt-1.5 text-sm leading-6" style={{ color: "#5c6b78" }}>
        {option.mood}
      </p>
      <div className="mt-3 flex flex-wrap gap-1.5">
        {option.palette.map((token) => (
          <span
            key={`${option.id}-${token}`}
            className="inline-flex h-4 w-4 rounded-full border"
            style={{ background: token, borderColor: "#d7dee6" }}
            title={token}
          />
        ))}
      </div>
    </button>
  );
}

export function RootPage() {
  const [activePreviewId, setActivePreviewId] = useState<PreviewId>("portal");
  const activePreview =
    previewOptions.find(({ id }) => id === activePreviewId) ??
    previewOptions[0];

  return (
    <main
      className="min-h-screen px-4 py-5 sm:px-6 lg:px-8"
      style={{
        background:
          "radial-gradient(circle at 15% 0%, #ffffff 0%, #edf2f7 45%, #e8edf3 100%)",
      }}
    >
      <div className="mx-auto grid max-w-[1480px] gap-5 xl:grid-cols-[320px_minmax(0,1fr)]">
        <aside
          className="h-fit rounded-2xl border bg-white p-5"
          style={{ borderColor: "#d8e1e8" }}
        >
          <p
            className="text-[11px] uppercase tracking-[0.3em]"
            style={{ color: "#7d8d9d" }}
          >
            预览实验室
          </p>
          <h1 className="mt-3 text-3xl font-semibold tracking-tight text-[#34495e]">
            ACMRank 风格预览
          </h1>
          <p className="mt-3 text-sm leading-7 text-[#5c6b78]">
            这轮只比较页面结构，不比较主色板。底色统一黑白灰，保留 Luogu
            风格主题色和绿色
            <span className="mx-1 font-medium text-[#2ea043]">
              Daily New AC
            </span>
            热力图。
          </p>

          <div className="mt-5 space-y-3">
            {previewOptions.map((option) => (
              <PreviewButton
                key={option.id}
                option={option}
                active={option.id === activePreviewId}
                onSelect={setActivePreviewId}
              />
            ))}
          </div>

          <div
            className="mt-5 rounded-xl border px-4 py-3"
            style={{ borderColor: "#d8e1e8", background: "#f8fbff" }}
          >
            <p
              className="text-xs uppercase tracking-[0.24em]"
              style={{ color: "#8191a0" }}
            >
              当前推荐
            </p>
            <p className="mt-2 text-sm leading-6 text-[#5c6b78]">
              {activePreview.recommendation}
            </p>
          </div>
        </aside>

        {activePreviewId === "portal" ? <PortalPreview /> : null}
        {activePreviewId === "profile" ? <ProfilePreview /> : null}
        {activePreviewId === "ranking" ? <RankingPreview /> : null}
      </div>
    </main>
  );
}
