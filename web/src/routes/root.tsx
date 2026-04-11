import { type CSSProperties, type ReactNode } from "react";

type IconName =
  | "home"
  | "rank"
  | "user"
  | "sync"
  | "heatmap"
  | "settings"
  | "search"
  | "bell"
  | "clock"
  | "arrow";

const colors = {
  pageBg: "#e8f1ef",
  shellBg: "#f7fbfa",
  panelBg: "#ffffff",
  panelSoft: "#f0f7f5",
  border: "#d9e7e2",
  text: "#243330",
  textMuted: "#5c716c",
  textSoft: "#8aa09b",
  accent: "#4aa695",
  accentDeep: "#3c8f7f",
  accentGlow: "rgba(74, 166, 149, 0.18)",
  warm: "#e9c95f",
  warmSoft: "#f6eed0",
  danger: "#d35f5f",
  heatmap: ["#ffffff", "#ebf7e8", "#cfe9d1", "#78c27d", "#2ea043"],
} as const;

const navItems = [
  { icon: "home" as const, label: "总览" },
  { icon: "rank" as const, label: "排行榜" },
  { icon: "user" as const, label: "个人页" },
  { icon: "sync" as const, label: "同步" },
  { icon: "heatmap" as const, label: "热力图" },
  { icon: "settings" as const, label: "设置" },
];

const actionItems = [
  { label: "绑定账号", detail: "新增 Codeforces / AtCoder / 洛谷" },
  { label: "发起同步", detail: "拉取最新 AC 与 rating 快照" },
  { label: "查看榜单", detail: "对比昨日变化与平台拆分" },
];

const platformMix = [
  { label: "Codeforces", solved: 634, share: 0.52, color: "#4aa695" },
  { label: "AtCoder", solved: 428, share: 0.35, color: "#e9c95f" },
  { label: "洛谷", solved: 154, share: 0.13, color: "#64b6d8" },
];

const rankingRows = [
  { rank: "01", user: "treneneno", score: "2476", delta: "+46" },
  { rank: "02", user: "ice", score: "2431", delta: "+19" },
  { rank: "03", user: "sherry", score: "2388", delta: "+11" },
  { rank: "04", user: "frost", score: "2340", delta: "-6" },
  { rank: "05", user: "lina", score: "2302", delta: "+3" },
];

const activityRows = [
  {
    time: "2026-04-11 10:32",
    title: "Codeforces 主链路同步完成",
    detail: "新增 7 题，回填 2 个 rating",
    delta: "+7",
  },
  {
    time: "2026-04-11 09:10",
    title: "AtCoder 聚合结果刷新",
    detail: "first_ac_at 更新 4 条",
    delta: "+4",
  },
  {
    time: "2026-04-10 23:59",
    title: "排行榜快照已生成",
    detail: "SCNU Rating 较昨日 +46",
    delta: "+46",
  },
];

const syncPlanRows = [
  { label: "Codeforces", value: "今天 18:00 自动同步" },
  { label: "AtCoder", value: "Cookie 主链路正常" },
  { label: "洛谷", value: "明天 08:30 拉取资料快照" },
];

const monthBars = [18, 12, 36, 44, 16, 0, 20, 14, 24, 26, 40, 52] as const;
const ratingTrend = [2260, 2294, 2312, 2340, 2368, 2384, 2418, 2476] as const;
const cfTrend = [14, 18, 19, 24, 21, 26, 28] as const;
const atTrend = [8, 7, 11, 13, 12, 15, 16] as const;

const heatmapWeeks = [
  [0, 1, 0, 2, 0, 1, 3],
  [1, 0, 2, 3, 1, 0, 0],
  [2, 3, 4, 1, 0, 1, 2],
  [0, 0, 1, 2, 3, 4, 2],
  [1, 2, 0, 0, 2, 3, 1],
  [3, 4, 2, 1, 0, 1, 0],
  [2, 1, 3, 2, 4, 2, 1],
] as const;

function buildLineGeometry(
  values: readonly number[],
  width: number,
  height: number,
) {
  const padding = 14;
  const min = Math.min(...values);
  const max = Math.max(...values);
  const range = max - min || 1;
  const step = (width - padding * 2) / Math.max(values.length - 1, 1);

  const points = values.map((value, index) => {
    const x = padding + index * step;
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

function Icon({
  name,
  className,
  strokeWidth = 1.8,
}: {
  name: IconName;
  className?: string;
  strokeWidth?: number;
}) {
  const props = {
    fill: "none",
    stroke: "currentColor",
    strokeLinecap: "round" as const,
    strokeLinejoin: "round" as const,
    strokeWidth,
  };

  return (
    <svg
      viewBox="0 0 24 24"
      className={className ?? "h-5 w-5"}
      aria-hidden="true"
    >
      {name === "home" ? (
        <>
          <path {...props} d="M4 11.5 12 5l8 6.5" />
          <path {...props} d="M6.5 10.5V19h11v-8.5" />
        </>
      ) : null}
      {name === "rank" ? (
        <>
          <path {...props} d="M5 19V9" />
          <path {...props} d="M12 19V5" />
          <path {...props} d="M19 19v-7" />
        </>
      ) : null}
      {name === "user" ? (
        <>
          <circle {...props} cx="12" cy="8" r="3.2" />
          <path {...props} d="M6 19c1.5-2.6 4-4 6-4s4.5 1.4 6 4" />
        </>
      ) : null}
      {name === "sync" ? (
        <>
          <path {...props} d="M7 8a6 6 0 0 1 10.2-1.9L19 8" />
          <path {...props} d="M17 16a6 6 0 0 1-10.2 1.9L5 16" />
          <path {...props} d="M19 8h-4" />
          <path {...props} d="M5 16h4" />
        </>
      ) : null}
      {name === "heatmap" ? (
        <>
          <rect {...props} x="5" y="5" width="5" height="5" rx="1" />
          <rect {...props} x="14" y="5" width="5" height="5" rx="1" />
          <rect {...props} x="5" y="14" width="5" height="5" rx="1" />
          <rect {...props} x="14" y="14" width="5" height="5" rx="1" />
        </>
      ) : null}
      {name === "settings" ? (
        <>
          <circle {...props} cx="12" cy="12" r="3.2" />
          <path {...props} d="M12 4v2.2" />
          <path {...props} d="M12 17.8V20" />
          <path {...props} d="m4.9 6.2 1.6 1.2" />
          <path {...props} d="m17.5 16.6 1.6 1.2" />
          <path {...props} d="M4 12h2.2" />
          <path {...props} d="M17.8 12H20" />
          <path {...props} d="m4.9 17.8 1.6-1.2" />
          <path {...props} d="m17.5 7.4 1.6-1.2" />
        </>
      ) : null}
      {name === "search" ? (
        <>
          <circle {...props} cx="11" cy="11" r="5" />
          <path {...props} d="m18 18 2.5 2.5" />
        </>
      ) : null}
      {name === "bell" ? (
        <>
          <path {...props} d="M8 17h8l-1.2-1.8V11a4.8 4.8 0 0 0-9.6 0v4.2Z" />
          <path {...props} d="M10 19a2 2 0 0 0 4 0" />
        </>
      ) : null}
      {name === "clock" ? (
        <>
          <circle {...props} cx="12" cy="12" r="8" />
          <path {...props} d="M12 7.5V12l3 2" />
        </>
      ) : null}
      {name === "arrow" ? (
        <>
          <path {...props} d="M5 12h13" />
          <path {...props} d="m14 7 4 5-4 5" />
        </>
      ) : null}
    </svg>
  );
}

function Card({
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
      className={`rounded-[28px] border p-4 shadow-[0_16px_36px_rgba(60,92,84,0.08)] sm:p-5 ${className ?? ""}`}
      style={{
        background: colors.panelBg,
        borderColor: colors.border,
        ...style,
      }}
    >
      {children}
    </article>
  );
}

function SectionMeta({ title, action }: { title: string; action?: string }) {
  return (
    <div className="mb-4 flex items-center justify-between gap-3">
      <h2 className="text-2xl font-semibold tracking-tight sm:text-3xl">
        {title}
      </h2>
      {action ? (
        <span
          className="rounded-full px-3 py-1 text-xs font-semibold"
          style={{ background: colors.panelSoft, color: colors.accentDeep }}
        >
          {action}
        </span>
      ) : null}
    </div>
  );
}

function SearchBar() {
  return (
    <div
      className="flex h-12 min-w-[280px] flex-1 items-center gap-3 rounded-2xl border px-4"
      style={{
        borderColor: colors.border,
        background: colors.panelSoft,
        color: colors.textSoft,
      }}
    >
      <Icon name="search" className="h-4 w-4" />
      <span className="text-sm">搜索用户、题号或比赛 ID</span>
    </div>
  );
}

function Sidebar() {
  return (
    <aside className="relative flex justify-center">
      <div
        className="relative min-h-[calc(100vh-96px)] w-[76px] rounded-[34px] border p-2"
        style={{
          background:
            "linear-gradient(180deg, #69b9ac 0%, #53ac9b 45%, #4aa695 100%)",
          borderColor: "#5ab2a0",
          boxShadow: "0 18px 38px rgba(62, 146, 128, 0.24)",
        }}
      >
        <div className="mt-2 flex justify-center">
          <span className="inline-flex h-11 w-11 items-center justify-center rounded-2xl bg-white/22 text-white">
            <Icon name="heatmap" className="h-5 w-5" />
          </span>
        </div>

        <nav className="mt-9 space-y-3">
          {navItems.map((item, index) => (
            <button
              key={item.label}
              type="button"
              title={item.label}
              className="flex h-11 w-full items-center justify-center rounded-2xl transition-colors"
              style={{
                background: index === 0 ? "#ffffff" : "rgba(255,255,255,0.12)",
                color: index === 0 ? colors.accentDeep : "#edf9f6",
              }}
            >
              <Icon name={item.icon} className="h-5 w-5" />
            </button>
          ))}
        </nav>
      </div>

      <div
        className="pointer-events-none absolute -left-1 top-28 h-16 w-4 rounded-r-full"
        style={{ background: "rgba(255,255,255,0.72)" }}
      />
      <div
        className="pointer-events-none absolute -left-1 bottom-28 h-16 w-4 rounded-r-full"
        style={{ background: "rgba(255,255,255,0.72)" }}
      />
    </aside>
  );
}

function AreaChart({
  title,
  values,
  tabs,
}: {
  title: string;
  values: readonly number[];
  tabs: string[];
}) {
  const { line, area } = buildLineGeometry(values, 360, 188);

  return (
    <Card className="h-full">
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-2xl font-semibold tracking-tight sm:text-3xl">
          {title}
        </h2>
        <div className="flex gap-2 text-xs">
          {tabs.map((tab, index) => (
            <span
              key={tab}
              className="rounded-full px-3 py-1"
              style={{
                background: index === 0 ? colors.panelSoft : "transparent",
                color: index === 0 ? colors.accentDeep : colors.textSoft,
              }}
            >
              {tab}
            </span>
          ))}
        </div>
      </div>

      <div className="mt-5">
        <svg viewBox="0 0 360 188" className="h-44 w-full">
          {[48, 92, 136].map((y) => (
            <line
              key={y}
              x1="14"
              x2="346"
              y1={y}
              y2={y}
              stroke={colors.border}
              strokeDasharray="6 6"
            />
          ))}
          <path d={area} fill={colors.accentGlow} />
          <polyline
            fill="none"
            points={line}
            stroke={colors.accent}
            strokeWidth="3.5"
            strokeLinecap="round"
          />
        </svg>
      </div>

      <div
        className="mt-2 flex justify-between text-xs"
        style={{ color: colors.textSoft }}
      >
        {["16", "17", "18", "19", "20", "21", "22", "23"].map((day) => (
          <span key={day}>{day}</span>
        ))}
      </div>
    </Card>
  );
}

function Sparkline({
  title,
  subtitle,
  value,
  values,
  secondaryValues,
}: {
  title: string;
  subtitle: string;
  value: string;
  values: readonly number[];
  secondaryValues?: readonly number[];
}) {
  const { line, area } = buildLineGeometry(values, 280, 92);
  const secondary = secondaryValues
    ? buildLineGeometry(secondaryValues, 280, 92)
    : null;

  return (
    <Card className="h-full">
      <div className="flex items-center justify-between gap-3">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight sm:text-3xl">
            {title}
          </h2>
          <p className="mt-1 text-sm" style={{ color: colors.textSoft }}>
            {subtitle}
          </p>
        </div>
      </div>
      <div className="mt-4">
        <svg viewBox="0 0 280 92" className="h-20 w-full">
          <path d={area} fill={colors.accentGlow} />
          <polyline
            fill="none"
            points={line}
            stroke={colors.accent}
            strokeWidth="3"
            strokeLinecap="round"
          />
          {secondary ? (
            <polyline
              fill="none"
              points={secondary.line}
              stroke={colors.warm}
              strokeWidth="2.5"
              strokeLinecap="round"
            />
          ) : null}
        </svg>
      </div>
      <p className="mt-3 text-4xl font-semibold tracking-tight">{value}</p>
    </Card>
  );
}

function Heatmap() {
  return (
    <div className="grid grid-flow-col grid-rows-7 gap-1.5 overflow-x-auto">
      {heatmapWeeks.flatMap((week, weekIndex) =>
        week.map((value, dayIndex) => (
          <div
            key={`${weekIndex}-${dayIndex}`}
            className="h-4 w-4 rounded-[4px] border"
            style={{
              background: colors.heatmap[value],
              borderColor: colors.border,
            }}
          />
        )),
      )}
    </div>
  );
}

function HeatmapCard() {
  return (
    <Card className="h-full">
      <div className="flex items-center justify-between gap-3">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight sm:text-3xl">
            Daily New AC
          </h2>
          <p className="mt-1 text-sm" style={{ color: colors.textSoft }}>
            GitHub 风格绿色热力图
          </p>
        </div>
        <span
          className="rounded-full px-3 py-1 text-xs font-semibold"
          style={{ background: colors.panelSoft, color: colors.accentDeep }}
        >
          连续 12 天活跃
        </span>
      </div>

      <div
        className="mt-5 rounded-2xl border p-4"
        style={{ background: colors.panelSoft, borderColor: colors.border }}
      >
        <div className="flex items-center justify-between text-xs">
          <span style={{ color: colors.textSoft }}>最近 7 周</span>
          <span style={{ color: colors.accentDeep }}>今日新增 12 题</span>
        </div>
        <div className="mt-4">
          <Heatmap />
        </div>
      </div>
    </Card>
  );
}

function PlatformBars() {
  return (
    <Card className="h-full">
      <div className="mb-4 flex items-center justify-between gap-3">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight sm:text-3xl">
            平台分布
          </h2>
          <p className="mt-1 text-sm" style={{ color: colors.textSoft }}>
            按平台累计已通过题数
          </p>
        </div>
        <span style={{ color: colors.textSoft }} className="text-xs">
          本学期
        </span>
      </div>

      <div className="grid grid-cols-12 items-end gap-2">
        {monthBars.map((value, index) => (
          <div
            key={index}
            className="h-28 rounded-2xl"
            style={{ background: "#edf4f2" }}
          >
            <div
              className="w-full rounded-2xl"
              style={{
                height: `${Math.max(value, 8)}%`,
                marginTop: `${100 - Math.max(value, 8)}%`,
                background: index % 3 === 1 ? colors.warm : colors.accent,
              }}
            />
          </div>
        ))}
      </div>

      <div
        className="mt-2 grid grid-cols-12 text-[10px]"
        style={{ color: colors.textSoft }}
      >
        {[
          "Jan",
          "Feb",
          "Mar",
          "Apr",
          "May",
          "Jun",
          "Jul",
          "Aug",
          "Sep",
          "Oct",
          "Nov",
          "Dec",
        ].map((month) => (
          <span key={month} className="text-center">
            {month}
          </span>
        ))}
      </div>

      <div className="mt-4 space-y-3">
        {platformMix.map((row) => (
          <div key={row.label} className="space-y-1.5">
            <div className="flex items-center justify-between text-sm">
              <span>{row.label}</span>
              <span style={{ color: colors.textMuted }}>{row.solved}</span>
            </div>
            <div
              className="h-2 overflow-hidden rounded-full"
              style={{ background: "#e7efed" }}
            >
              <div
                className="h-full rounded-full"
                style={{
                  width: `${row.share * 100}%`,
                  background: row.color,
                }}
              />
            </div>
          </div>
        ))}
      </div>
    </Card>
  );
}

function SyncPlanCard() {
  return (
    <Card className="h-full">
      <SectionMeta title="同步计划" action="今天" />
      <div className="space-y-3">
        {syncPlanRows.map((row) => (
          <div
            key={row.label}
            className="rounded-2xl border px-4 py-3"
            style={{ background: colors.panelSoft, borderColor: colors.border }}
          >
            <p className="text-sm font-medium">{row.label}</p>
            <p className="mt-1 text-xs" style={{ color: colors.textSoft }}>
              {row.value}
            </p>
          </div>
        ))}
      </div>
    </Card>
  );
}

function ActivityCard() {
  return (
    <Card className="h-full">
      <SectionMeta title="最新动态" action="查看全部" />
      <div className="space-y-3">
        {activityRows.map((row) => (
          <div
            key={`${row.time}-${row.title}`}
            className="grid gap-3 rounded-2xl border px-4 py-3 sm:grid-cols-[150px_minmax(0,1fr)_72px]"
            style={{ background: colors.panelSoft, borderColor: colors.border }}
          >
            <p className="text-xs" style={{ color: colors.textSoft }}>
              {row.time}
            </p>
            <div>
              <p className="text-sm font-medium">{row.title}</p>
              <p className="mt-1 text-xs" style={{ color: colors.textSoft }}>
                {row.detail}
              </p>
            </div>
            <p
              className="text-right text-sm font-semibold"
              style={{ color: colors.accentDeep }}
            >
              {row.delta}
            </p>
          </div>
        ))}
      </div>
    </Card>
  );
}

function RankingCard() {
  return (
    <Card className="h-full">
      <SectionMeta title="排行榜" action="SCNU Rating" />
      <div className="space-y-2.5">
        {rankingRows.map((row) => (
          <div
            key={row.rank}
            className="grid grid-cols-[42px_minmax(0,1fr)_72px_56px] items-center gap-3 rounded-2xl border px-3 py-3 text-sm"
            style={{
              background: row.rank === "01" ? "#eef8f5" : colors.panelSoft,
              borderColor: row.rank === "01" ? "#c9e8df" : colors.border,
            }}
          >
            <span style={{ color: colors.textMuted }}>{row.rank}</span>
            <span className="truncate">{row.user}</span>
            <span className="text-right">{row.score}</span>
            <span
              className="text-right"
              style={{
                color: row.delta.startsWith("+")
                  ? colors.accentDeep
                  : colors.danger,
              }}
            >
              {row.delta}
            </span>
          </div>
        ))}
      </div>
    </Card>
  );
}

function ProfileOverviewCard() {
  return (
    <Card className="h-full">
      <div className="grid gap-4 lg:grid-cols-[1.1fr_0.95fr]">
        <div>
          <div className="mb-4 flex items-center justify-between gap-3">
            <div>
              <h1 className="text-4xl font-semibold tracking-tight sm:text-5xl">
                我的主页
              </h1>
              <p className="mt-2 text-sm" style={{ color: colors.textMuted }}>
                一个真正属于 ACMRank 的整页用户 Dashboard。
              </p>
            </div>
            <button
              type="button"
              className="inline-flex items-center gap-2 rounded-full px-4 py-2 text-sm font-semibold"
              style={{ background: colors.panelSoft, color: colors.accentDeep }}
            >
              <span>公开页</span>
              <Icon name="arrow" className="h-4 w-4" />
            </button>
          </div>

          <div className="grid gap-3 sm:grid-cols-[1fr_0.92fr]">
            <div
              className="rounded-[26px] border p-4 text-white"
              style={{
                borderColor: "#67bcaf",
                background:
                  "linear-gradient(145deg, #5ab6a6 0%, #4aa695 68%, #e9c95f 100%)",
                boxShadow: "0 18px 30px rgba(74,166,149,0.18)",
              }}
            >
              <p className="text-xs text-white/80">Public Dashboard</p>
              <h2 className="mt-3 text-2xl font-semibold">treneneno</h2>
              <p className="mt-1 text-sm text-white/85">
                陈某某 / SCNU ICPC Team
              </p>
              <div className="mt-5 grid grid-cols-3 gap-3 text-xs text-white/85">
                <div>
                  <p>SCNU Rating</p>
                  <p className="mt-1 text-lg font-semibold text-white">2476</p>
                </div>
                <div>
                  <p>总过题</p>
                  <p className="mt-1 text-lg font-semibold text-white">1216</p>
                </div>
                <div>
                  <p>校内排名</p>
                  <p className="mt-1 text-lg font-semibold text-white">#1</p>
                </div>
              </div>
            </div>

            <div
              className="rounded-[26px] border p-4"
              style={{
                background: colors.panelSoft,
                borderColor: colors.border,
              }}
            >
              <p className="text-sm" style={{ color: colors.textMuted }}>
                当前概览
              </p>
              <p className="mt-2 text-4xl font-semibold">+23</p>
              <p className="mt-1 text-sm" style={{ color: colors.textSoft }}>
                本周新增 AC
              </p>

              <div className="mt-5 space-y-2">
                {["Codeforces", "AtCoder", "洛谷"].map((platform) => (
                  <span
                    key={platform}
                    className="inline-flex rounded-full px-3 py-1 text-xs font-semibold"
                    style={{
                      marginRight: "0.5rem",
                      background: colors.panelBg,
                      color: colors.textMuted,
                      border: `1px solid ${colors.border}`,
                    }}
                  >
                    {platform}
                  </span>
                ))}
              </div>

              <button
                type="button"
                className="mt-6 rounded-full px-4 py-2 text-sm font-semibold text-white"
                style={{ background: colors.accent }}
              >
                查看完整过题列表
              </button>
            </div>
          </div>
        </div>

        <div>
          <p className="text-sm" style={{ color: colors.textMuted }}>
            你现在最常做的事
          </p>
          <div className="mt-3 grid gap-3 sm:grid-cols-3 lg:grid-cols-1 xl:grid-cols-3">
            {actionItems.map((item) => (
              <button
                key={item.label}
                type="button"
                className="rounded-[24px] border p-4 text-left transition-[transform,box-shadow] duration-200 hover:-translate-y-0.5 hover:shadow-[0_16px_28px_rgba(60,92,84,0.10)]"
                style={{
                  background: colors.panelSoft,
                  borderColor: colors.border,
                  cursor: "pointer",
                }}
              >
                <span
                  className="inline-flex h-10 w-10 items-center justify-center rounded-full text-white"
                  style={{ background: colors.accent }}
                >
                  <Icon name="arrow" className="h-4 w-4" />
                </span>
                <p className="mt-5 text-base font-semibold">{item.label}</p>
                <p
                  className="mt-2 text-xs leading-6"
                  style={{ color: colors.textSoft }}
                >
                  {item.detail}
                </p>
              </button>
            ))}
          </div>

          <div
            className="mt-3 rounded-[24px] border px-4 py-3"
            style={{ background: "#fbf7e7", borderColor: "#efdf9f" }}
          >
            <p className="text-sm font-medium">本周目标</p>
            <p
              className="mt-1 text-xs leading-6"
              style={{ color: colors.textMuted }}
            >
              AtCoder 再补 5 题，Codeforces 训练分保持增长，热力图不要断。
            </p>
          </div>
        </div>
      </div>
    </Card>
  );
}

function HeaderBar() {
  return (
    <header
      className="flex flex-wrap items-center justify-between gap-3 rounded-[28px] border px-4 py-3"
      style={{ background: colors.panelBg, borderColor: colors.border }}
    >
      <SearchBar />
      <div className="flex items-center gap-2">
        <span
          className="inline-flex h-10 w-10 items-center justify-center rounded-full border"
          style={{ borderColor: colors.border, color: colors.textMuted }}
        >
          <Icon name="settings" className="h-4 w-4" />
        </span>
        <span
          className="inline-flex h-10 w-10 items-center justify-center rounded-full border"
          style={{ borderColor: colors.border, color: colors.textMuted }}
        >
          <Icon name="clock" className="h-4 w-4" />
        </span>
        <span
          className="inline-flex h-10 w-10 items-center justify-center rounded-full border"
          style={{ borderColor: colors.border, color: colors.textMuted }}
        >
          <Icon name="bell" className="h-4 w-4" />
        </span>
        <span
          className="rounded-full px-3 py-2 text-xs font-semibold"
          style={{ background: colors.panelSoft, color: colors.accentDeep }}
        >
          treneneno
        </span>
      </div>
    </header>
  );
}

export function RootPage() {
  return (
    <main
      className="min-h-screen p-3 sm:p-4 lg:p-5"
      style={{
        background:
          "radial-gradient(circle at 10% 0%, #f8fffd 0%, #edf6f4 46%, #e6efed 100%)",
        color: colors.text,
      }}
    >
      <section
        className="w-full rounded-[36px] border p-3 shadow-[0_26px_60px_rgba(48,82,74,0.14)] sm:p-4 lg:min-h-[calc(100vh-2.5rem)] lg:p-5"
        style={{ background: colors.shellBg, borderColor: colors.border }}
      >
        <div className="grid gap-4 xl:grid-cols-[92px_minmax(0,1fr)]">
          <Sidebar />

          <div className="space-y-4">
            <HeaderBar />

            <div className="grid gap-4 xl:grid-cols-[1.55fr_0.95fr]">
              <ProfileOverviewCard />
              <AreaChart
                title="SCNU Rating"
                values={ratingTrend}
                tabs={["Week", "Month", "Year"]}
              />
            </div>

            <div className="grid gap-4 xl:grid-cols-[1fr_1fr_1.2fr]">
              <Sparkline
                title="训练状态"
                subtitle="AT / CF 本周训练强度"
                value="AT 16 / CF 28"
                values={cfTrend}
                secondaryValues={atTrend}
              />
              <HeatmapCard />
              <PlatformBars />
            </div>

            <div className="grid gap-4 xl:grid-cols-[0.95fr_1.35fr_0.9fr]">
              <SyncPlanCard />
              <ActivityCard />
              <RankingCard />
            </div>
          </div>
        </div>
      </section>
    </main>
  );
}
