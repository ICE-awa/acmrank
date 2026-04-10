import { useState, type CSSProperties, type ReactNode } from "react";

type PreviewId = "ink-stone" | "paper-column" | "steel-frame";

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

const previewOptions: PreviewOption[] = [
  {
    id: "ink-stone",
    label: "Luogu Portal",
    eyebrow: "Luogu palette / portal layout",
    mood: "更像首页门户，模块活跃，适合放公告、排行榜、个人入口。",
    recommendation: "如果你希望首页看起来更有生气，这一版最适合继续深化。",
    accent: "#3498db",
    palette: ["#34495e", "#e9eaec", "#ffffff", "#5c5c5c", "#3498db"],
  },
  {
    id: "paper-column",
    label: "Luogu Profile",
    eyebrow: "Luogu palette / profile layout",
    mood: "更像用户个人页，左侧人物信息，右侧图表和题目视图。",
    recommendation: "如果你想先把用户公开页做顺，这一版会更接近正式主产品。",
    accent: "#3498db",
    palette: ["#34495e", "#e9eaec", "#ffffff", "#5c5c5c", "#3498db"],
  },
  {
    id: "steel-frame",
    label: "Luogu Rankings",
    eyebrow: "Luogu palette / ranking layout",
    mood: "更像榜单和数据总览页，适合核心排行榜与趋势展示。",
    recommendation:
      "如果你想把评分、变化值和榜单做成站点主视觉，这一版更合适。",
    accent: "#3498db",
    palette: ["#34495e", "#e9eaec", "#ffffff", "#5c5c5c", "#3498db"],
  },
] as const;

const metricCards = [
  { label: "SCNU Rating", value: "2476", detail: "较昨日 +46" },
  { label: "Today New AC", value: "12", detail: "每日 00:00 清零" },
  { label: "Verified Accounts", value: "7", detail: "CF / AT / 洛谷" },
] as const;

const leaderboardRows = [
  { rank: "01", user: "treneneno", score: "2476", delta: "+46" },
  { rank: "02", user: "ice", score: "2431", delta: "+19" },
  { rank: "03", user: "sherry", score: "2388", delta: "+11" },
  { rank: "04", user: "frost", score: "2340", delta: "-6" },
] as const;

const problemRows = [
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
] as const;

const awardRows = [
  "ICPC EC Final 2025 银奖",
  "广东省赛 2025 金奖",
  "校队选拔训练营 2026 A 组",
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
  serif: '"Source Han Serif SC", "Noto Serif SC", "Songti SC", serif',
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

function Stage({
  children,
  className,
  style,
}: {
  children: ReactNode;
  className?: string;
  style?: CSSProperties;
}) {
  return (
    <section
      className={`rounded-[32px] border shadow-[0_24px_80px_rgba(0,0,0,0.12)] ${className ?? ""}`}
      style={style}
    >
      {children}
    </section>
  );
}

function MetricGrid({
  tokens,
  compact = false,
}: {
  tokens: Tokens;
  compact?: boolean;
}) {
  return (
    <div
      className={`grid gap-3 ${compact ? "sm:grid-cols-3" : "lg:grid-cols-3"}`}
    >
      {metricCards.map((card) => (
        <article
          key={card.label}
          className="rounded-[20px] border p-4"
          style={{ background: tokens.panelAlt, borderColor: tokens.line }}
        >
          <p
            className="text-[11px] uppercase tracking-[0.24em]"
            style={{
              color:
                card.label === "SCNU Rating" ? tokens.accent : tokens.textMuted,
            }}
          >
            {card.label}
          </p>
          <p
            className="mt-3 text-3xl font-semibold"
            style={{ color: tokens.text }}
          >
            {card.value}
          </p>
          <p className="mt-2 text-sm" style={{ color: tokens.textSoft }}>
            {card.detail}
          </p>
        </article>
      ))}
    </div>
  );
}

function RankingList({ tokens }: { tokens: Tokens }) {
  return (
    <div className="space-y-3">
      {leaderboardRows.map((row) => (
        <div
          key={row.rank}
          className="grid grid-cols-[40px_minmax(0,1fr)_72px_56px] items-center gap-3 rounded-[16px] border px-3 py-3 text-sm"
          style={{
            background: tokens.panelAlt,
            borderColor: tokens.line,
            color: tokens.text,
          }}
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
            style={{
              color: row.delta.startsWith("+")
                ? tokens.accent
                : tokens.textSoft,
            }}
          >
            {row.delta}
          </span>
        </div>
      ))}
    </div>
  );
}

function ProblemTable({ tokens }: { tokens: Tokens }) {
  return (
    <div className="space-y-3">
      {problemRows.map((problem) => (
        <div
          key={problem.id}
          className="grid gap-3 rounded-[16px] border px-4 py-3 sm:grid-cols-[1fr_84px_148px]"
          style={{
            background: tokens.panelAlt,
            borderColor: tokens.line,
            color: tokens.text,
          }}
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

function Heatmap({ tokens }: { tokens: Tokens }) {
  return (
    <div className="grid grid-flow-col grid-rows-7 gap-2 overflow-x-auto">
      {heatmapWeeks.flatMap((week, weekIndex) =>
        week.map((value, dayIndex) => (
          <div
            key={`${weekIndex}-${dayIndex}`}
            className="h-[18px] w-[18px] rounded-[5px] border"
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
  const { points, line, area } = buildChartGeometry(ratingTrend, 420, 220);

  return (
    <svg viewBox="0 0 420 220" className="h-56 w-full">
      {[56, 104, 152].map((y) => (
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
        strokeWidth="3.5"
        strokeLinecap="round"
      />
      {points.map(({ x, y }) => (
        <circle key={`${x}-${y}`} cx={x} cy={y} r="4" fill={tokens.chart} />
      ))}
    </svg>
  );
}

function Awards({ tokens }: { tokens: Tokens }) {
  return (
    <div className="space-y-3">
      {awardRows.map((award) => (
        <div
          key={award}
          className="rounded-[16px] border px-4 py-3 text-sm"
          style={{
            background: tokens.panelAlt,
            borderColor: tokens.line,
            color: tokens.text,
          }}
        >
          {award}
        </div>
      ))}
    </div>
  );
}

function InkStonePreview() {
  const tokens: Tokens = {
    page: "#e9eaec",
    panel: "#ffffff",
    panelAlt: "#f5f8fb",
    line: "#d8e1e8",
    text: "#34495e",
    textMuted: "#5c5c5c",
    textSoft: "#7a7a7a",
    accent: "#3498db",
    accentSoft: "rgba(52, 152, 219, 0.12)",
    chart: "#3498db",
    heatmap: ["#ffffff", "#ebf5fc", "#cde4f5", "#7dbbe7", "#3498db"],
  };

  return (
    <Stage
      className="border-neutral-300"
      style={{
        background: tokens.page,
        color: tokens.text,
        fontFamily: fontStacks.sans,
      }}
    >
      <div className="bg-[#3498db] px-6 py-4 text-white">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex flex-wrap items-center gap-4">
            <span className="text-lg font-semibold">ACMRank</span>
            <span className="text-sm text-white/85">Portal</span>
            <span className="text-sm text-white/85">Rankings</span>
            <span className="text-sm text-white/85">Profiles</span>
          </div>
          <span className="rounded-full bg-white/16 px-3 py-1 text-xs">
            Luogu palette
          </span>
        </div>
      </div>
      <div className="p-6 sm:p-8">
        <header className="border-b pb-6" style={{ borderColor: tokens.line }}>
          <p
            className="text-[11px] uppercase tracking-[0.3em]"
            style={{ color: tokens.accent }}
          >
            Luogu Portal
          </p>
          <div className="mt-4 grid gap-5 xl:grid-cols-[1.15fr_0.85fr]">
            <div>
              <h2 className="text-4xl font-semibold tracking-[-0.04em] sm:text-5xl">
                保留洛谷这套蓝灰白颜色，但把首页做得更像“活的门户”。
              </h2>
              <p
                className="mt-4 max-w-3xl text-sm leading-7"
                style={{ color: tokens.textMuted }}
              >
                这一版不再做沉闷的大卡片拼盘，而是参考洛谷那种更有流动感的首页。蓝色做主轴，白卡承载内容，灰底负责留白和呼吸。
              </p>
              <div className="mt-5 flex flex-wrap gap-2 text-xs">
                {[
                  ["Problems", "#e74c3c"],
                  ["Trainings", "#f39c12"],
                  ["Contests", "#9b59b6"],
                  ["Teams", "#3498db"],
                  ["Discuss", "#34495e"],
                ].map(([label, color]) => (
                  <span
                    key={label}
                    className="rounded-full px-3 py-1.5 text-white"
                    style={{ background: color }}
                  >
                    {label}
                  </span>
                ))}
              </div>
            </div>
            <MetricGrid tokens={tokens} compact />
          </div>
        </header>

        <div className="mt-6 grid gap-5 xl:grid-cols-[1.4fr_0.95fr]">
          <div className="space-y-5">
            <article
              className="rounded-[26px] border p-5"
              style={{ background: tokens.panel, borderColor: tokens.line }}
            >
              <div className="flex flex-wrap items-end justify-between gap-4">
                <div>
                  <p
                    className="text-[11px] uppercase tracking-[0.24em]"
                    style={{ color: tokens.accent }}
                  >
                    Campus Portal
                  </p>
                  <h3 className="mt-3 text-3xl font-semibold">
                    公告、趋势、榜单和个人入口可以共存，而且不显得堵。
                  </h3>
                </div>
                <div className="flex flex-wrap gap-2 text-xs">
                  {["Announcement", "Ranking", "Profile", "Awards"].map(
                    (item) => (
                      <span
                        key={item}
                        className="rounded-full border px-3 py-1.5"
                        style={{
                          borderColor: tokens.line,
                          background: tokens.panelAlt,
                          color: tokens.textMuted,
                        }}
                      >
                        {item}
                      </span>
                    ),
                  )}
                </div>
              </div>
              <div className="mt-6 grid gap-4 lg:grid-cols-[1.2fr_0.8fr]">
                <div
                  className="rounded-[22px] border p-4"
                  style={{
                    background: tokens.panelAlt,
                    borderColor: tokens.line,
                  }}
                >
                  <div className="mb-4 flex items-end justify-between">
                    <p
                      className="text-[11px] uppercase tracking-[0.24em]"
                      style={{ color: tokens.accent }}
                    >
                      SCNU Rating Curve
                    </p>
                    <span
                      className="text-sm"
                      style={{ color: tokens.textSoft }}
                    >
                      首页趋势卡
                    </span>
                  </div>
                  <TrendChart tokens={tokens} />
                </div>
                <div
                  className="rounded-[22px] border p-4"
                  style={{
                    background: tokens.panelAlt,
                    borderColor: tokens.line,
                  }}
                >
                  <div className="mb-4 flex items-end justify-between">
                    <p
                      className="text-[11px] uppercase tracking-[0.24em]"
                      style={{ color: tokens.accent }}
                    >
                      Daily New AC
                    </p>
                    <span
                      className="text-sm"
                      style={{ color: tokens.textSoft }}
                    >
                      热力图入口
                    </span>
                  </div>
                  <Heatmap tokens={tokens} />
                </div>
              </div>
            </article>

            <div className="grid gap-5 lg:grid-cols-[0.95fr_1.05fr]">
              <article
                className="rounded-[26px] border p-5"
                style={{ background: tokens.panel, borderColor: tokens.line }}
              >
                <p
                  className="text-[11px] uppercase tracking-[0.24em]"
                  style={{ color: tokens.accent }}
                >
                  Ranking Module
                </p>
                <h4 className="mt-3 text-2xl font-semibold">Campus ranking</h4>
                <div className="mt-5">
                  <RankingList tokens={tokens} />
                </div>
              </article>
              <article
                className="rounded-[26px] border p-5"
                style={{ background: tokens.panel, borderColor: tokens.line }}
              >
                <p
                  className="text-[11px] uppercase tracking-[0.24em]"
                  style={{ color: tokens.accent }}
                >
                  Problem Entry
                </p>
                <h4 className="mt-3 text-2xl font-semibold">
                  Latest solved set
                </h4>
                <div className="mt-5">
                  <ProblemTable tokens={tokens} />
                </div>
              </article>
            </div>
          </div>

          <div className="space-y-5">
            <article
              className="rounded-[26px] border p-5"
              style={{ background: tokens.panel, borderColor: tokens.line }}
            >
              <p
                className="text-[11px] uppercase tracking-[0.24em]"
                style={{ color: tokens.accent }}
              >
                Why This Works
              </p>
              <ul
                className="mt-4 space-y-3 text-sm leading-7"
                style={{ color: tokens.textMuted }}
              >
                <li>首页更像站点入口，而不是单一数据面板。</li>
                <li>多色只出现在导航入口，主内容仍由蓝灰白控制。</li>
                <li>这一版最能解决“死气沉沉”的问题。</li>
              </ul>
            </article>
            <article
              className="rounded-[26px] border p-5"
              style={{ background: tokens.panel, borderColor: tokens.line }}
            >
              <p
                className="text-[11px] uppercase tracking-[0.24em]"
                style={{ color: tokens.accent }}
              >
                ICPC Awards
              </p>
              <h4 className="mt-3 text-2xl font-semibold">Award history</h4>
              <div className="mt-5">
                <Awards tokens={tokens} />
              </div>
            </article>
          </div>
        </div>
      </div>
    </Stage>
  );
}

function PaperColumnPreview() {
  const tokens: Tokens = {
    page: "#f5f5f5",
    panel: "#ffffff",
    panelAlt: "#f7f9fb",
    line: "#dddddd",
    text: "#34495e",
    textMuted: "#525252",
    textSoft: "#737373",
    accent: "#3498db",
    accentSoft: "rgba(52, 152, 219, 0.12)",
    chart: "#3498db",
    heatmap: ["#ffffff", "#ebf5fc", "#cde4f5", "#7dbbe7", "#3498db"],
  };

  return (
    <Stage
      className="border-neutral-300"
      style={{
        background: tokens.page,
        color: tokens.text,
        fontFamily: fontStacks.sans,
      }}
    >
      <div className="bg-[#34495e] px-6 py-3 text-white">
        <div className="flex flex-wrap items-center gap-5 text-sm">
          <span className="font-semibold">ACMRank</span>
          <span className="text-white/75">Profile</span>
          <span className="text-white/75">Problem View</span>
          <span className="rounded-full bg-[#3498db] px-3 py-1">
            Public Page
          </span>
        </div>
      </div>
      <div className="p-6 sm:p-8">
        <header className="border-b pb-6" style={{ borderColor: tokens.line }}>
          <p
            className="text-[11px] uppercase tracking-[0.3em]"
            style={{ color: tokens.accent }}
          >
            Luogu Profile
          </p>
          <div className="mt-4 grid gap-5 xl:grid-cols-[1.05fr_0.95fr]">
            <div>
              <h2 className="text-4xl font-semibold leading-tight sm:text-5xl">
                同一套洛谷配色，换成更正常的“个人页优先”布局。
              </h2>
              <p
                className="mt-4 max-w-3xl text-sm leading-7"
                style={{ color: tokens.textMuted }}
              >
                这一版把用户本人放回页面中心。左侧是身份、账号和奖项，右侧是折线图、热力图和题目视图，更接近真正会落地的公开用户页。
              </p>
            </div>
            <MetricGrid tokens={tokens} compact />
          </div>
        </header>

        <div className="mt-6 grid gap-5 xl:grid-cols-[0.88fr_1.12fr]">
          <article
            className="rounded-[26px] border p-5"
            style={{ background: tokens.panel, borderColor: tokens.line }}
          >
            <p
              className="text-[11px] uppercase tracking-[0.24em]"
              style={{ color: tokens.textMuted }}
            >
              Editorial Note
            </p>
            <h3 className="mt-3 text-3xl leading-tight">
              左侧人物信息固定，右侧专注展示训练结果，这样读起来会顺很多。
            </h3>
            <ul
              className="mt-5 space-y-3 text-sm leading-7"
              style={{ color: tokens.textMuted }}
            >
              <li>信息入口减少，注意力更集中在单个用户身上。</li>
              <li>图表和题目列表的关系更自然，不像首页模块在抢位置。</li>
              <li>如果主产品核心是公开个人页，这一版最值得继续做。</li>
            </ul>
            <div
              className="mt-6 border-t pt-5"
              style={{ borderColor: tokens.line }}
            >
              <p
                className="text-[11px] uppercase tracking-[0.24em]"
                style={{ color: tokens.textMuted }}
              >
                ICPC Awards
              </p>
              <div className="mt-4">
                <Awards tokens={tokens} />
              </div>
            </div>
          </article>

          <div className="space-y-5">
            <article
              className="rounded-[26px] border p-5"
              style={{ background: tokens.panel, borderColor: tokens.line }}
            >
              <div className="flex flex-wrap items-end justify-between gap-4">
                <div>
                  <p
                    className="text-[11px] uppercase tracking-[0.24em]"
                    style={{ color: tokens.textMuted }}
                  >
                    Public Profile Layout
                  </p>
                  <h3 className="mt-3 text-3xl leading-tight">
                    更像真实用户页，而不是门户页缩小版。
                  </h3>
                </div>
                <div
                  className="flex flex-wrap gap-2 text-xs"
                  style={{ color: tokens.textMuted }}
                >
                  {["Profile", "Awards", "Problems", "Rating"].map((item) => (
                    <span
                      key={item}
                      className="rounded-full border px-3 py-1.5"
                      style={{
                        borderColor: tokens.line,
                        background: tokens.panelAlt,
                      }}
                    >
                      {item}
                    </span>
                  ))}
                </div>
              </div>
              <div className="mt-6 grid gap-4 lg:grid-cols-[1.08fr_0.92fr]">
                <div
                  className="rounded-[22px] border p-4"
                  style={{
                    background: tokens.panelAlt,
                    borderColor: tokens.line,
                  }}
                >
                  <div className="mb-4 flex items-end justify-between">
                    <p
                      className="text-[11px] uppercase tracking-[0.24em]"
                      style={{ color: tokens.textMuted }}
                    >
                      SCNU Rating Curve
                    </p>
                    <span
                      className="text-sm"
                      style={{ color: tokens.textSoft }}
                    >
                      profile curve
                    </span>
                  </div>
                  <TrendChart tokens={tokens} />
                </div>
                <div
                  className="rounded-[22px] border p-4"
                  style={{
                    background: tokens.panelAlt,
                    borderColor: tokens.line,
                  }}
                >
                  <div className="mb-4 flex items-end justify-between">
                    <p
                      className="text-[11px] uppercase tracking-[0.24em]"
                      style={{ color: tokens.textMuted }}
                    >
                      Daily Heatmap
                    </p>
                    <span
                      className="text-sm"
                      style={{ color: tokens.textSoft }}
                    >
                      daily newly accepted
                    </span>
                  </div>
                  <Heatmap tokens={tokens} />
                </div>
              </div>
            </article>

            <div className="grid gap-5 lg:grid-cols-[0.88fr_1.12fr]">
              <article
                className="rounded-[26px] border p-5"
                style={{ background: tokens.panel, borderColor: tokens.line }}
              >
                <p
                  className="text-[11px] uppercase tracking-[0.24em]"
                  style={{ color: tokens.textMuted }}
                >
                  Leaderboard
                </p>
                <h4 className="mt-3 text-2xl font-semibold">Campus ranking</h4>
                <div className="mt-5">
                  <RankingList tokens={tokens} />
                </div>
              </article>
              <article
                className="rounded-[26px] border p-5"
                style={{ background: tokens.panel, borderColor: tokens.line }}
              >
                <p
                  className="text-[11px] uppercase tracking-[0.24em]"
                  style={{ color: tokens.textMuted }}
                >
                  Problem Ledger
                </p>
                <h4 className="mt-3 text-2xl font-semibold">Solved problems</h4>
                <div className="mt-5">
                  <ProblemTable tokens={tokens} />
                </div>
              </article>
            </div>
          </div>
        </div>
      </div>
    </Stage>
  );
}

function SteelFramePreview() {
  const tokens: Tokens = {
    page: "#e9eaec",
    panel: "#ffffff",
    panelAlt: "#f5f8fb",
    line: "#d8e1e8",
    text: "#34495e",
    textMuted: "#525252",
    textSoft: "#737373",
    accent: "#3498db",
    accentSoft: "rgba(52, 152, 219, 0.12)",
    chart: "#3498db",
    heatmap: ["#ffffff", "#ebf5fc", "#cde4f5", "#7dbbe7", "#3498db"],
  };

  return (
    <Stage
      className="border-neutral-400"
      style={{
        background: tokens.page,
        color: tokens.text,
        fontFamily: fontStacks.sans,
      }}
    >
      <div className="bg-[#3498db] px-6 py-4 text-white">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex flex-wrap items-center gap-4">
            <span className="text-lg font-semibold">ACMRank</span>
            <span className="text-sm text-white/85">Rankings</span>
            <span className="text-sm text-white/85">Trend</span>
            <span className="text-sm text-white/85">Heatmap</span>
          </div>
          <span className="rounded-full bg-white/16 px-3 py-1 text-xs">
            Luogu palette
          </span>
        </div>
      </div>
      <div className="p-6 sm:p-8">
        <header
          className="rounded-[26px] border p-5"
          style={{ background: "#ffffff", borderColor: tokens.line }}
        >
          <p
            className="text-[11px] uppercase tracking-[0.3em]"
            style={{ color: tokens.accent }}
          >
            Luogu Rankings
          </p>
          <div className="mt-4 grid gap-5 xl:grid-cols-[1.05fr_0.95fr]">
            <div>
              <h2 className="text-4xl font-semibold tracking-[-0.04em] sm:text-5xl">
                用洛谷这套颜色，直接把榜单和趋势做成站点主视觉。
              </h2>
              <p
                className="mt-4 max-w-3xl text-sm leading-7"
                style={{ color: tokens.textMuted }}
              >
                如果首页的核心价值是排行榜和成长曲线，那就不必再塞太多入口。把视觉重心压到榜单、折线图和热力图，会更直接。
              </p>
            </div>
            <MetricGrid tokens={tokens} compact />
          </div>
        </header>

        <div className="mt-5 grid gap-5 xl:grid-cols-[1.08fr_0.92fr]">
          <div className="space-y-5">
            <article
              className="rounded-[26px] border p-5"
              style={{ background: tokens.panel, borderColor: tokens.line }}
            >
              <div className="flex flex-wrap items-end justify-between gap-4">
                <div>
                  <p
                    className="text-[11px] uppercase tracking-[0.24em]"
                    style={{ color: tokens.textMuted }}
                  >
                    Unified Product Shell
                  </p>
                  <h3 className="mt-3 text-3xl font-semibold">
                    把评分、变化值和榜单做成主舞台，页面会更有冲劲。
                  </h3>
                </div>
                <div
                  className="flex flex-wrap gap-2 text-xs"
                  style={{ color: tokens.textMuted }}
                >
                  {["Rankings", "Rating", "Heatmap", "Recent"].map((item) => (
                    <span
                      key={item}
                      className="rounded-full border px-3 py-1.5"
                      style={{
                        borderColor: tokens.line,
                        background: tokens.panelAlt,
                      }}
                    >
                      {item}
                    </span>
                  ))}
                </div>
              </div>
              <div className="mt-6 grid gap-4 lg:grid-cols-[1.05fr_0.95fr]">
                <div
                  className="rounded-[22px] border p-4"
                  style={{
                    background: tokens.panelAlt,
                    borderColor: tokens.line,
                  }}
                >
                  <div className="mb-4 flex items-end justify-between">
                    <p
                      className="text-[11px] uppercase tracking-[0.24em]"
                      style={{ color: tokens.textMuted }}
                    >
                      Top Ranking
                    </p>
                    <span
                      className="text-sm"
                      style={{ color: tokens.textSoft }}
                    >
                      SCNU Rating
                    </span>
                  </div>
                  <RankingList tokens={tokens} />
                </div>
                <div
                  className="rounded-[22px] border p-4"
                  style={{
                    background: tokens.panelAlt,
                    borderColor: tokens.line,
                  }}
                >
                  <div className="mb-4 flex items-end justify-between">
                    <p
                      className="text-[11px] uppercase tracking-[0.24em]"
                      style={{ color: tokens.textMuted }}
                    >
                      Daily Heatmap
                    </p>
                    <span
                      className="text-sm"
                      style={{ color: tokens.textSoft }}
                    >
                      daily new AC
                    </span>
                  </div>
                  <Heatmap tokens={tokens} />
                </div>
              </div>
            </article>

            <article
              className="rounded-[26px] border p-5"
              style={{ background: tokens.panel, borderColor: tokens.line }}
            >
              <div className="mb-4 flex items-end justify-between">
                <div>
                  <p
                    className="text-[11px] uppercase tracking-[0.24em]"
                    style={{ color: tokens.textMuted }}
                  >
                    SCNU Rating Curve
                  </p>
                  <h4 className="mt-3 text-2xl font-semibold">Trend view</h4>
                </div>
                <span className="text-sm" style={{ color: tokens.textSoft }}>
                  growth focus
                </span>
              </div>
              <TrendChart tokens={tokens} />
            </article>
          </div>

          <div className="space-y-5">
            <article
              className="rounded-[26px] border p-5"
              style={{ background: tokens.panel, borderColor: tokens.line }}
            >
              <p
                className="text-[11px] uppercase tracking-[0.24em]"
                style={{ color: tokens.textMuted }}
              >
                Solved Problems
              </p>
              <h4 className="mt-3 text-2xl font-semibold">Problem table</h4>
              <div className="mt-5">
                <ProblemTable tokens={tokens} />
              </div>
            </article>
            <article
              className="rounded-[26px] border p-5"
              style={{ background: tokens.panel, borderColor: tokens.line }}
            >
              <p
                className="text-[11px] uppercase tracking-[0.24em]"
                style={{ color: tokens.textMuted }}
              >
                ICPC Awards
              </p>
              <h4 className="mt-3 text-2xl font-semibold">Award history</h4>
              <div className="mt-5">
                <Awards tokens={tokens} />
              </div>
            </article>
            <article
              className="rounded-[26px] border p-5"
              style={{ background: tokens.panel, borderColor: tokens.line }}
            >
              <p
                className="text-[11px] uppercase tracking-[0.24em]"
                style={{ color: tokens.textMuted }}
              >
                Why This Works
              </p>
              <ul
                className="mt-4 space-y-3 text-sm leading-7"
                style={{ color: tokens.textMuted }}
              >
                <li>把站点价值压缩成几个高权重模块，气质会更利落。</li>
                <li>适合首页直接展示 SCNU Rating 排行榜时使用。</li>
                <li>如果你最在意“核心指标看起来够不够强”，这一版最好用。</li>
              </ul>
            </article>
          </div>
        </div>
      </div>
    </Stage>
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
      className="w-full rounded-[22px] border px-4 py-4 text-left transition"
      style={{
        borderColor: active ? option.accent : "#2f2f2f",
        background: active ? "#262626" : "#171717",
      }}
    >
      <p
        className="text-[11px] uppercase tracking-[0.24em]"
        style={{ color: active ? option.accent : "#737373" }}
      >
        {option.eyebrow}
      </p>
      <h2 className="mt-3 text-xl font-semibold text-neutral-50">
        {option.label}
      </h2>
      <p className="mt-2 text-sm leading-6 text-neutral-400">{option.mood}</p>
      <div className="mt-4 flex flex-wrap gap-2">
        {option.palette.map((token) => (
          <span
            key={`${option.id}-${token}`}
            className="inline-flex items-center gap-2 rounded-full border border-neutral-700 px-2.5 py-1 text-xs text-neutral-400"
          >
            <span
              className="h-2.5 w-2.5 rounded-full border border-neutral-600"
              style={{ background: token }}
            />
            {token}
          </span>
        ))}
      </div>
    </button>
  );
}

export function RootPage() {
  const [activePreviewId, setActivePreviewId] =
    useState<PreviewId>("ink-stone");
  const activePreview =
    previewOptions.find(({ id }) => id === activePreviewId) ??
    previewOptions[0];

  return (
    <main className="min-h-screen bg-[#0a0a0a] px-4 py-5 text-neutral-50 sm:px-6 lg:px-8">
      <div className="mx-auto grid max-w-[1480px] gap-5 xl:grid-cols-[320px_minmax(0,1fr)]">
        <aside className="h-fit rounded-[32px] border border-neutral-800 bg-[#111111] p-5">
          <p className="text-[11px] uppercase tracking-[0.3em] text-neutral-500">
            Style Sandbox
          </p>
          <h1 className="mt-4 text-3xl font-semibold tracking-tight">
            ACMRank Frontend Preview Lab
          </h1>
          <p className="mt-3 text-sm leading-7 text-neutral-400">
            这一轮不再比较不同配色了，直接锁定你认可的洛谷色板，只比较 3
            种更正常的页面结构。你选定后，我再把风格约束写入
            <code className="mx-1 rounded bg-neutral-800 px-1.5 py-0.5 text-xs text-neutral-200">
              AGENTS.md
            </code>
            ，然后回退这次预览改动。
          </p>
          <div className="mt-6 space-y-3">
            {previewOptions.map((option) => (
              <PreviewButton
                key={option.id}
                option={option}
                active={option.id === activePreviewId}
                onSelect={setActivePreviewId}
              />
            ))}
          </div>
          <div className="mt-6 rounded-[22px] border border-neutral-800 bg-[#171717] p-4">
            <p className="text-xs uppercase tracking-[0.24em] text-neutral-500">
              Current Read
            </p>
            <p className="mt-3 text-sm leading-6 text-neutral-300">
              {activePreview.recommendation}
            </p>
          </div>
        </aside>

        {activePreviewId === "ink-stone" ? <InkStonePreview /> : null}
        {activePreviewId === "paper-column" ? <PaperColumnPreview /> : null}
        {activePreviewId === "steel-frame" ? <SteelFramePreview /> : null}
      </div>
    </main>
  );
}
