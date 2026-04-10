import { useState, type CSSProperties, type ReactNode } from "react";

type PreviewId = "signal-lab" | "archive-ledger" | "trackside-pulse";

type PreviewOption = {
  id: PreviewId;
  label: string;
  eyebrow: string;
  mood: string;
  recommendation: string;
  palette: Array<{ label: string; value: string }>;
};

const previewOptions: PreviewOption[] = [
  {
    id: "signal-lab",
    label: "Signal Lab",
    eyebrow: "Midnight dashboard",
    mood: "冷静、理性、偏运营控制台，适合做数据感很强的公开页。",
    recommendation: "如果你想让 ACMRank 更像训练数据中枢，这套最稳。",
    palette: [
      { label: "Primary", value: "#4ade80" },
      { label: "Secondary", value: "#38bdf8" },
      { label: "Surface", value: "#08111f" },
      { label: "Text", value: "#e2f7ef" },
    ],
  },
  {
    id: "archive-ledger",
    label: "Archive Ledger",
    eyebrow: "Editorial paper",
    mood: "更像校史档案与成绩总册，公开展示的辨识度最高。",
    recommendation: "如果你希望它更像“竞赛档案馆”，这套最有气质。",
    palette: [
      { label: "Primary", value: "#1f6d5a" },
      { label: "Secondary", value: "#b65c33" },
      { label: "Surface", value: "#f6efe2" },
      { label: "Text", value: "#2d241c" },
    ],
  },
  {
    id: "trackside-pulse",
    label: "Trackside Pulse",
    eyebrow: "Athletic poster",
    mood: "更像竞赛海报和训练战报，动势强，适合突出成长与速度。",
    recommendation: "如果你想让首页更有冲劲和记忆点，这套最鲜明。",
    palette: [
      { label: "Primary", value: "#fb7185" },
      { label: "Secondary", value: "#f59e0b" },
      { label: "Surface", value: "#130d16" },
      { label: "Text", value: "#fff7ed" },
    ],
  },
] as const;

const statCards = [
  { label: "SCNU Rating", value: "2476", detail: "较昨日 +46" },
  { label: "Today New AC", value: "12", detail: "00:00 后重新计数" },
  { label: "Verified Accounts", value: "7", detail: "CF / AT / 洛谷 已聚合" },
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
  signal:
    '"Space Grotesk", "IBM Plex Sans", "Noto Sans SC", "PingFang SC", sans-serif',
  archiveSans: '"IBM Plex Sans", "Noto Sans SC", "PingFang SC", sans-serif',
  archiveSerif: '"Source Han Serif SC", "Noto Serif SC", "Songti SC", serif',
  pulse: '"Sora", "Avenir Next", "Noto Sans SC", "PingFang SC", sans-serif',
  mono: '"IBM Plex Mono", "JetBrains Mono", "SFMono-Regular", monospace',
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

  const polyline = points.map(({ x, y }) => `${x},${y}`).join(" ");
  const first = points[0];
  const last = points.at(-1) ?? first;
  const area = [
    `M ${first?.x ?? padding} ${height - padding}`,
    ...points.map(({ x, y }) => `L ${x} ${y}`),
    `L ${last?.x ?? padding} ${height - padding}`,
    "Z",
  ].join(" ");

  return { points, polyline, area };
}

function TrendChart({
  stroke,
  fill,
  grid,
  label,
  text,
}: {
  stroke: string;
  fill: string;
  grid: string;
  label: string;
  text: string;
}) {
  const { points, polyline, area } = buildChartGeometry(ratingTrend, 420, 220);

  return (
    <figure className="space-y-4">
      <div className="flex items-end justify-between">
        <figcaption
          className="text-xs uppercase tracking-[0.24em]"
          style={{ color: label }}
        >
          SCNU Rating Curve
        </figcaption>
        <span className="text-sm" style={{ color: text }}>
          近 12 次快照
        </span>
      </div>
      <svg viewBox="0 0 420 220" className="h-56 w-full">
        {[56, 104, 152].map((line) => (
          <line
            key={line}
            x1="16"
            x2="404"
            y1={line}
            y2={line}
            stroke={grid}
            strokeDasharray="6 8"
          />
        ))}
        <path d={area} fill={fill} />
        <polyline
          fill="none"
          points={polyline}
          stroke={stroke}
          strokeWidth="4"
          strokeLinecap="round"
        />
        {points.map(({ x, y }) => (
          <circle key={`${x}-${y}`} cx={x} cy={y} r="4.5" fill={stroke} />
        ))}
      </svg>
    </figure>
  );
}

function Heatmap({
  shades,
  cellBorder,
}: {
  shades: [string, string, string, string, string];
  cellBorder: string;
}) {
  return (
    <div className="grid grid-flow-col grid-rows-7 gap-2">
      {heatmapWeeks.flatMap((week, weekIndex) =>
        week.map((value, dayIndex) => (
          <div
            key={`${weekIndex}-${dayIndex}`}
            className="h-5 w-5 rounded-[6px] border"
            style={{
              backgroundColor: shades[value],
              borderColor: cellBorder,
            }}
            title={`Week ${weekIndex + 1}, day ${dayIndex + 1}: ${value} new AC`}
          />
        )),
      )}
    </div>
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
      className={`w-full rounded-[24px] border px-4 py-4 text-left transition ${
        active
          ? "border-white/35 bg-white/14 shadow-[0_20px_60px_rgba(15,23,42,0.25)]"
          : "border-white/10 bg-white/5 hover:border-white/20 hover:bg-white/9"
      }`}
    >
      <p className="text-[11px] uppercase tracking-[0.24em] text-white/55">
        {option.eyebrow}
      </p>
      <div className="mt-3 flex items-start justify-between gap-3">
        <div>
          <h2 className="text-xl font-semibold text-white">{option.label}</h2>
          <p className="mt-2 text-sm leading-6 text-white/72">{option.mood}</p>
        </div>
      </div>
      <div className="mt-4 flex flex-wrap gap-2">
        {option.palette.map((token) => (
          <span
            key={`${option.id}-${token.label}`}
            className="inline-flex items-center gap-2 rounded-full border border-white/10 px-2.5 py-1 text-xs text-white/78"
          >
            <span
              className="h-2.5 w-2.5 rounded-full"
              style={{ backgroundColor: token.value }}
              aria-hidden="true"
            />
            {token.label}
          </span>
        ))}
      </div>
    </button>
  );
}

function StageShell({
  children,
  style,
}: {
  children: ReactNode;
  style: CSSProperties;
}) {
  return (
    <section
      className="min-h-[980px] rounded-[34px] border border-white/10 shadow-[0_28px_90px_rgba(2,6,23,0.3)]"
      style={style}
    >
      {children}
    </section>
  );
}

function SignalLabPreview() {
  return (
    <StageShell
      style={{
        background:
          "radial-gradient(circle at top right, rgba(56,189,248,0.2), transparent 28%), linear-gradient(180deg, #09111f 0%, #06111c 48%, #040914 100%)",
        color: "#e2f7ef",
        fontFamily: fontStacks.signal,
      }}
    >
      <div className="relative overflow-hidden rounded-[34px]">
        <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(rgba(56,189,248,0.08)_1px,transparent_1px),linear-gradient(90deg,rgba(56,189,248,0.08)_1px,transparent_1px)] bg-[size:40px_40px]" />
        <div className="pointer-events-none absolute inset-x-0 top-0 h-64 bg-[radial-gradient(circle_at_top,rgba(74,222,128,0.18),transparent_72%)]" />

        <div className="relative p-6 sm:p-8">
          <header className="flex flex-col gap-5 border-b border-white/10 pb-6 lg:flex-row lg:items-end lg:justify-between">
            <div className="space-y-4">
              <span className="inline-flex rounded-full border border-sky-300/20 bg-sky-300/10 px-3 py-1 text-[11px] uppercase tracking-[0.28em] text-sky-100/85">
                Signal Lab Preview
              </span>
              <div className="space-y-3">
                <h2 className="max-w-3xl text-4xl font-semibold tracking-[-0.03em] sm:text-5xl">
                  Contest Signals, Clean Facts
                </h2>
                <p className="max-w-3xl text-sm leading-7 text-[#b7d7d0] sm:text-base">
                  这套更偏数据控制台：深色底、冷色高亮、信息密度高，适合把排行榜、个人页指标、
                  热力图和题目事实做成统一的分析面板。
                </p>
              </div>
            </div>

            <div className="grid gap-3 sm:grid-cols-3">
              {statCards.map((card) => (
                <article
                  key={card.label}
                  className="min-w-[170px] rounded-[22px] border border-white/10 bg-black/25 p-4 backdrop-blur"
                >
                  <p className="text-[11px] uppercase tracking-[0.24em] text-[#8ac6c1]">
                    {card.label}
                  </p>
                  <p className="mt-3 text-3xl font-semibold text-white">
                    {card.value}
                  </p>
                  <p className="mt-2 text-sm text-[#9cc7bf]">{card.detail}</p>
                </article>
              ))}
            </div>
          </header>

          <div className="mt-6 grid gap-5 xl:grid-cols-[1.45fr_0.95fr]">
            <div className="space-y-5">
              <section className="rounded-[28px] border border-white/10 bg-[#08192a]/88 p-5 shadow-[0_20px_60px_rgba(3,8,19,0.45)]">
                <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
                  <div>
                    <p
                      className="text-[11px] uppercase tracking-[0.24em] text-[#56d0ff]"
                      style={{ fontFamily: fontStacks.mono }}
                    >
                      Public Profile / u.treneneno
                    </p>
                    <h3 className="mt-3 text-3xl font-semibold text-white sm:text-[2.6rem]">
                      SCNU training archive with operational clarity
                    </h3>
                  </div>
                  <div className="flex flex-wrap gap-2 text-xs">
                    {["Rankings", "Profile", "Platforms", "ICPC"].map(
                      (item) => (
                        <span
                          key={item}
                          className="rounded-full border border-white/10 bg-white/5 px-3 py-1.5 text-[#c6e3dd]"
                        >
                          {item}
                        </span>
                      ),
                    )}
                  </div>
                </div>
                <div className="mt-6 grid gap-4 lg:grid-cols-[1.2fr_0.8fr]">
                  <div className="rounded-[24px] border border-sky-300/14 bg-[#07131f] p-4">
                    <TrendChart
                      stroke="#38bdf8"
                      fill="rgba(56,189,248,0.16)"
                      grid="rgba(148,163,184,0.22)"
                      label="#7dd3fc"
                      text="#d7f0ea"
                    />
                  </div>
                  <div className="rounded-[24px] border border-emerald-300/12 bg-[#08141d] p-4">
                    <div className="flex items-end justify-between">
                      <div>
                        <p
                          className="text-[11px] uppercase tracking-[0.24em] text-[#7ef0b8]"
                          style={{ fontFamily: fontStacks.mono }}
                        >
                          Daily New AC
                        </p>
                        <h4 className="mt-2 text-2xl font-semibold text-white">
                          Heatmap
                        </h4>
                      </div>
                      <p className="text-sm text-[#aad7cf]">只统计每日新 AC</p>
                    </div>
                    <div className="mt-5 overflow-x-auto">
                      <Heatmap
                        shades={[
                          "#06111c",
                          "#10304a",
                          "#155e75",
                          "#0f766e",
                          "#22c55e",
                        ]}
                        cellBorder="rgba(148, 163, 184, 0.16)"
                      />
                    </div>
                  </div>
                </div>
              </section>

              <section className="grid gap-5 lg:grid-cols-[0.95fr_1.05fr]">
                <article className="rounded-[28px] border border-white/10 bg-[#07111d]/90 p-5">
                  <div className="flex items-end justify-between">
                    <div>
                      <p className="text-[11px] uppercase tracking-[0.24em] text-[#56d0ff]">
                        Leaderboard Slice
                      </p>
                      <h4 className="mt-2 text-2xl font-semibold text-white">
                        Campus ranking
                      </h4>
                    </div>
                    <span className="rounded-full border border-white/10 px-3 py-1 text-xs text-[#a8d6cf]">
                      23:59 baseline delta
                    </span>
                  </div>
                  <div className="mt-5 space-y-3">
                    {leaderboardRows.map((row) => (
                      <div
                        key={row.rank}
                        className="grid grid-cols-[48px_minmax(0,1fr)_88px_72px] items-center gap-3 rounded-[18px] border border-white/8 bg-white/4 px-3 py-3"
                      >
                        <span
                          className="text-sm text-[#8ec7c2]"
                          style={{ fontFamily: fontStacks.mono }}
                        >
                          {row.rank}
                        </span>
                        <span className="truncate text-sm text-white">
                          {row.user}
                        </span>
                        <span className="text-right text-sm text-[#dff9f0]">
                          {row.score}
                        </span>
                        <span
                          className={`text-right text-sm ${row.delta.startsWith("+") ? "text-[#7ef0b8]" : "text-[#fca5a5]"}`}
                        >
                          {row.delta}
                        </span>
                      </div>
                    ))}
                  </div>
                </article>

                <article className="rounded-[28px] border border-white/10 bg-[#07111d]/90 p-5">
                  <div className="flex items-end justify-between gap-4">
                    <div>
                      <p className="text-[11px] uppercase tracking-[0.24em] text-[#56d0ff]">
                        Accepted Facts
                      </p>
                      <h4 className="mt-2 text-2xl font-semibold text-white">
                        Problem stream
                      </h4>
                    </div>
                    <span className="text-xs text-[#9dc9c3]">
                      AC only / no full submission history
                    </span>
                  </div>
                  <div className="mt-5 space-y-3">
                    {problemRows.map((problem) => (
                      <div
                        key={problem.id}
                        className="rounded-[18px] border border-white/8 bg-white/4 px-4 py-3"
                      >
                        <div className="flex flex-wrap items-center justify-between gap-3">
                          <div>
                            <p className="text-sm font-medium text-white">
                              {problem.id}
                            </p>
                            <p className="mt-1 text-xs text-[#a3cdc6]">
                              {problem.contest}
                            </p>
                          </div>
                          <div className="text-right">
                            <p className="text-sm text-[#dff9f0]">
                              {problem.rating}
                            </p>
                            <p className="mt-1 text-xs text-[#9ac6bf]">
                              {problem.time}
                            </p>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </article>
              </section>
            </div>

            <aside className="space-y-5">
              <article className="rounded-[28px] border border-white/10 bg-[#08131e]/92 p-5">
                <p className="text-[11px] uppercase tracking-[0.24em] text-[#7ef0b8]">
                  Design Read
                </p>
                <h4 className="mt-3 text-2xl font-semibold text-white">
                  Where this style works
                </h4>
                <ul className="mt-4 space-y-3 text-sm leading-7 text-[#b2d8d1]">
                  <li>
                    适合排行榜、个人页和平台视图共享一套深色数据面板语言。
                  </li>
                  <li>
                    对热力图、折线图、表格都很友好，后续扩展管理端也顺手。
                  </li>
                  <li>风险是学术档案感略弱，更偏“训练系统”而不是“校史馆”。</li>
                </ul>
              </article>

              <article className="rounded-[28px] border border-white/10 bg-[#08131e]/92 p-5">
                <div className="flex items-end justify-between">
                  <div>
                    <p className="text-[11px] uppercase tracking-[0.24em] text-[#56d0ff]">
                      ICPC Timeline
                    </p>
                    <h4 className="mt-2 text-2xl font-semibold text-white">
                      Award history
                    </h4>
                  </div>
                  <span className="text-xs text-[#9ac6bf]">
                    Public profile module
                  </span>
                </div>
                <div className="mt-5 space-y-3">
                  {awardRows.map((award) => (
                    <div
                      key={award}
                      className="rounded-[18px] border border-white/8 bg-white/4 px-4 py-3 text-sm text-[#dbeee9]"
                    >
                      {award}
                    </div>
                  ))}
                </div>
              </article>
            </aside>
          </div>
        </div>
      </div>
    </StageShell>
  );
}

function ArchiveLedgerPreview() {
  return (
    <StageShell
      style={{
        background:
          "linear-gradient(180deg, #f8f1e7 0%, #f5ede0 58%, #efe3d0 100%)",
        color: "#2d241c",
        fontFamily: fontStacks.archiveSans,
      }}
    >
      <div className="relative overflow-hidden rounded-[34px]">
        <div className="pointer-events-none absolute inset-x-0 top-0 h-56 bg-[radial-gradient(circle_at_top,rgba(182,92,51,0.18),transparent_70%)]" />
        <div className="pointer-events-none absolute inset-y-0 left-8 w-px bg-[#d9cdbb]" />

        <div className="relative p-6 sm:p-8">
          <header className="grid gap-5 border-b border-[#d9cdbb] pb-6 xl:grid-cols-[1.1fr_0.9fr]">
            <div>
              <p className="text-[11px] uppercase tracking-[0.32em] text-[#8b5d46]">
                Archive Ledger Preview
              </p>
              <h2
                className="mt-4 max-w-3xl text-4xl leading-tight sm:text-5xl"
                style={{ fontFamily: fontStacks.archiveSerif }}
              >
                训练档案像校史馆一样可靠，也像成绩总册一样耐看
              </h2>
              <p className="mt-4 max-w-3xl text-sm leading-7 text-[#5b4a3f] sm:text-base">
                这套强调“竞赛档案与公开陈列”的气质。它会让排行榜、个人页和奖项历史更像一套经过编排的刊物，
                不是冷冰冰的后台。
              </p>
            </div>

            <div className="grid gap-3 sm:grid-cols-3">
              {statCards.map((card) => (
                <article
                  key={card.label}
                  className="rounded-[24px] border border-[#d9cdbb] bg-[#fbf6ee] p-4 shadow-[0_18px_45px_rgba(91,74,63,0.08)]"
                >
                  <p className="text-[11px] uppercase tracking-[0.24em] text-[#826452]">
                    {card.label}
                  </p>
                  <p
                    className="mt-3 text-3xl leading-none text-[#1f6d5a]"
                    style={{ fontFamily: fontStacks.archiveSerif }}
                  >
                    {card.value}
                  </p>
                  <p className="mt-2 text-sm text-[#68574b]">{card.detail}</p>
                </article>
              ))}
            </div>
          </header>

          <div className="mt-6 grid gap-5 xl:grid-cols-[0.92fr_1.08fr]">
            <aside className="space-y-5">
              <article className="rounded-[30px] border border-[#d9cdbb] bg-[#fbf6ee] p-5">
                <p className="text-[11px] uppercase tracking-[0.24em] text-[#8b5d46]">
                  Curatorial Note
                </p>
                <h3
                  className="mt-3 text-3xl leading-tight text-[#2d241c]"
                  style={{ fontFamily: fontStacks.archiveSerif }}
                >
                  更像竞赛档案馆，而不是泛化的 SaaS 面板
                </h3>
                <ul className="mt-4 space-y-3 text-sm leading-7 text-[#5b4a3f]">
                  <li>奖项历史、人物信息、训练曲线会自然形成“展陈”感。</li>
                  <li>浅底深字更利于长时间阅读题目列表和排行详情。</li>
                  <li>风险是管理端若沿用同风格，需要额外克制信息密度。</li>
                </ul>
              </article>

              <article className="rounded-[30px] border border-[#d9cdbb] bg-[#faf3e8] p-5">
                <div className="flex items-end justify-between gap-4 border-b border-[#d9cdbb] pb-4">
                  <div>
                    <p className="text-[11px] uppercase tracking-[0.24em] text-[#8b5d46]">
                      ICPC Chronicle
                    </p>
                    <h4
                      className="mt-2 text-2xl text-[#2d241c]"
                      style={{ fontFamily: fontStacks.archiveSerif }}
                    >
                      Award history
                    </h4>
                  </div>
                  <span className="text-xs text-[#826452]">
                    public exhibition block
                  </span>
                </div>
                <div className="mt-4 space-y-4">
                  {awardRows.map((award, index) => (
                    <div
                      key={award}
                      className="grid grid-cols-[30px_minmax(0,1fr)] gap-4"
                    >
                      <div className="flex flex-col items-center">
                        <span className="mt-1 h-2.5 w-2.5 rounded-full bg-[#1f6d5a]" />
                        {index < awardRows.length - 1 ? (
                          <span className="mt-2 h-full w-px bg-[#d9cdbb]" />
                        ) : null}
                      </div>
                      <div className="rounded-[18px] border border-[#dfd3c3] bg-[#fffaf2] px-4 py-3 text-sm leading-6 text-[#4f4035]">
                        {award}
                      </div>
                    </div>
                  ))}
                </div>
              </article>
            </aside>

            <div className="space-y-5">
              <section className="rounded-[30px] border border-[#d9cdbb] bg-[#fffaf2] p-5">
                <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
                  <div>
                    <p className="text-[11px] uppercase tracking-[0.24em] text-[#8b5d46]">
                      Public Profile Spread
                    </p>
                    <h3
                      className="mt-3 text-3xl leading-tight text-[#2d241c] sm:text-[2.6rem]"
                      style={{ fontFamily: fontStacks.archiveSerif }}
                    >
                      让排行榜和个人页像一本被长期维护的训练总册
                    </h3>
                  </div>
                  <div className="flex flex-wrap gap-2 text-xs text-[#5b4a3f]">
                    {["Rankings", "Profiles", "Platforms", "Awards"].map(
                      (item) => (
                        <span
                          key={item}
                          className="rounded-full border border-[#d9cdbb] bg-[#f6ede0] px-3 py-1.5"
                        >
                          {item}
                        </span>
                      ),
                    )}
                  </div>
                </div>

                <div className="mt-6 grid gap-5 lg:grid-cols-[1.15fr_0.85fr]">
                  <div className="rounded-[24px] border border-[#d9cdbb] bg-[#faf3e8] p-4">
                    <TrendChart
                      stroke="#1f6d5a"
                      fill="rgba(31,109,90,0.12)"
                      grid="rgba(139,93,70,0.16)"
                      label="#8b5d46"
                      text="#5b4a3f"
                    />
                  </div>

                  <div className="rounded-[24px] border border-[#d9cdbb] bg-[#faf3e8] p-4">
                    <div className="flex items-end justify-between gap-3">
                      <div>
                        <p className="text-[11px] uppercase tracking-[0.24em] text-[#8b5d46]">
                          Daily Heatmap
                        </p>
                        <h4
                          className="mt-2 text-2xl text-[#2d241c]"
                          style={{ fontFamily: fontStacks.archiveSerif }}
                        >
                          每日新 AC
                        </h4>
                      </div>
                      <span className="text-xs text-[#6d5b4d]">
                        GitHub-like, but warmer
                      </span>
                    </div>
                    <div className="mt-5 overflow-x-auto">
                      <Heatmap
                        shades={[
                          "#f7efe3",
                          "#dde8dc",
                          "#b7d2c6",
                          "#6eab96",
                          "#1f6d5a",
                        ]}
                        cellBorder="rgba(91, 74, 63, 0.12)"
                      />
                    </div>
                  </div>
                </div>
              </section>

              <section className="grid gap-5 lg:grid-cols-[0.88fr_1.12fr]">
                <article className="rounded-[30px] border border-[#d9cdbb] bg-[#faf3e8] p-5">
                  <div className="flex items-end justify-between">
                    <div>
                      <p className="text-[11px] uppercase tracking-[0.24em] text-[#8b5d46]">
                        Ranking Table
                      </p>
                      <h4
                        className="mt-2 text-2xl text-[#2d241c]"
                        style={{ fontFamily: fontStacks.archiveSerif }}
                      >
                        Campus leaderboard
                      </h4>
                    </div>
                    <span className="text-xs text-[#6d5b4d]">
                      editorial density
                    </span>
                  </div>
                  <div className="mt-5 space-y-3">
                    {leaderboardRows.map((row) => (
                      <div
                        key={row.rank}
                        className="grid grid-cols-[42px_minmax(0,1fr)_78px_64px] items-center gap-3 border-b border-[#e1d8ca] pb-3 text-sm text-[#3c3028]"
                      >
                        <span className="text-[#8b5d46]">{row.rank}</span>
                        <span className="truncate">{row.user}</span>
                        <span className="text-right">{row.score}</span>
                        <span
                          className={`text-right ${row.delta.startsWith("+") ? "text-[#1f6d5a]" : "text-[#b65c33]"}`}
                        >
                          {row.delta}
                        </span>
                      </div>
                    ))}
                  </div>
                </article>

                <article className="rounded-[30px] border border-[#d9cdbb] bg-[#faf3e8] p-5">
                  <div className="flex items-end justify-between gap-4 border-b border-[#d9cdbb] pb-4">
                    <div>
                      <p className="text-[11px] uppercase tracking-[0.24em] text-[#8b5d46]">
                        Accepted Ledger
                      </p>
                      <h4
                        className="mt-2 text-2xl text-[#2d241c]"
                        style={{ fontFamily: fontStacks.archiveSerif }}
                      >
                        Problem register
                      </h4>
                    </div>
                    <span className="text-xs text-[#6d5b4d]">
                      只展示 AC 事实
                    </span>
                  </div>
                  <div className="mt-4 space-y-3">
                    {problemRows.map((problem) => (
                      <div
                        key={problem.id}
                        className="grid gap-3 rounded-[18px] border border-[#dfd3c3] bg-[#fffaf2] px-4 py-3 sm:grid-cols-[1fr_110px_160px]"
                      >
                        <div>
                          <p className="text-sm font-medium text-[#2d241c]">
                            {problem.id}
                          </p>
                          <p className="mt-1 text-xs text-[#6d5b4d]">
                            {problem.contest}
                          </p>
                        </div>
                        <p className="text-sm text-[#1f6d5a] sm:text-right">
                          {problem.rating}
                        </p>
                        <p className="text-sm text-[#5b4a3f] sm:text-right">
                          {problem.time}
                        </p>
                      </div>
                    ))}
                  </div>
                </article>
              </section>
            </div>
          </div>
        </div>
      </div>
    </StageShell>
  );
}

function TracksidePulsePreview() {
  return (
    <StageShell
      style={{
        background:
          "radial-gradient(circle at top left, rgba(251,113,133,0.26), transparent 24%), radial-gradient(circle at bottom right, rgba(245,158,11,0.2), transparent 28%), linear-gradient(180deg, #120d16 0%, #18101a 40%, #09090b 100%)",
        color: "#fff7ed",
        fontFamily: fontStacks.pulse,
      }}
    >
      <div className="relative overflow-hidden rounded-[34px]">
        <div className="pointer-events-none absolute -left-12 top-20 h-48 w-48 rounded-full bg-rose-400/16 blur-3xl" />
        <div className="pointer-events-none absolute right-0 top-0 h-64 w-64 rounded-full bg-amber-300/12 blur-3xl" />
        <div className="pointer-events-none absolute inset-x-0 top-0 h-32 bg-[linear-gradient(90deg,transparent,rgba(255,255,255,0.08),transparent)]" />

        <div className="relative p-6 sm:p-8">
          <header className="flex flex-col gap-6 border-b border-white/10 pb-6 xl:flex-row xl:items-end xl:justify-between">
            <div>
              <p className="text-[11px] uppercase tracking-[0.32em] text-rose-200/72">
                Trackside Pulse Preview
              </p>
              <h2 className="mt-4 max-w-4xl text-4xl font-semibold leading-tight tracking-[-0.03em] sm:text-5xl">
                把训练曲线做成一张有速度感的竞赛战报
              </h2>
              <p className="mt-4 max-w-3xl text-sm leading-7 text-orange-100/80 sm:text-base">
                这套更偏公开形象页：大数字、强对比、暖色高光和海报化排版，适合把成长感、
                冲榜氛围和个人辨识度放大。
              </p>
            </div>

            <div className="grid gap-3 sm:grid-cols-3">
              {statCards.map((card, index) => (
                <article
                  key={card.label}
                  className={`rounded-[26px] border px-4 py-4 backdrop-blur ${
                    index === 0
                      ? "border-rose-200/20 bg-rose-400/10"
                      : index === 1
                        ? "border-amber-200/20 bg-amber-300/10"
                        : "border-white/10 bg-white/5"
                  }`}
                >
                  <p className="text-[11px] uppercase tracking-[0.24em] text-orange-100/60">
                    {card.label}
                  </p>
                  <p className="mt-3 text-3xl font-semibold text-white">
                    {card.value}
                  </p>
                  <p className="mt-2 text-sm text-orange-50/70">
                    {card.detail}
                  </p>
                </article>
              ))}
            </div>
          </header>

          <section className="mt-6 rounded-[32px] border border-white/10 bg-black/16 p-5 shadow-[0_28px_80px_rgba(0,0,0,0.28)]">
            <div className="grid gap-5 xl:grid-cols-[1.08fr_0.92fr]">
              <div className="space-y-5">
                <div className="flex flex-wrap gap-2 text-xs">
                  {["Rankings", "Profiles", "Growth", "ICPC"].map((item) => (
                    <span
                      key={item}
                      className="rounded-full border border-white/10 bg-white/6 px-3 py-1.5 text-orange-50/75"
                    >
                      {item}
                    </span>
                  ))}
                </div>
                <div className="grid gap-5 lg:grid-cols-[0.84fr_1.16fr]">
                  <article className="rounded-[28px] border border-white/10 bg-[#1b121a]/88 p-5">
                    <p className="text-[11px] uppercase tracking-[0.28em] text-rose-200/68">
                      Profile Card
                    </p>
                    <h3 className="mt-4 text-4xl font-semibold tracking-[-0.04em] text-white">
                      treneneno
                    </h3>
                    <p className="mt-3 max-w-sm text-sm leading-7 text-orange-50/75">
                      Public profile that feels closer to a competition poster
                      than a generic dashboard.
                    </p>
                    <div className="mt-6 grid gap-3 sm:grid-cols-2">
                      <div className="rounded-[22px] border border-white/10 bg-white/5 p-4">
                        <p className="text-xs uppercase tracking-[0.24em] text-orange-50/55">
                          AT Training
                        </p>
                        <p className="mt-3 text-3xl font-semibold text-white">
                          2512
                        </p>
                      </div>
                      <div className="rounded-[22px] border border-white/10 bg-white/5 p-4">
                        <p className="text-xs uppercase tracking-[0.24em] text-orange-50/55">
                          CF Training
                        </p>
                        <p className="mt-3 text-3xl font-semibold text-white">
                          2440
                        </p>
                      </div>
                    </div>
                  </article>

                  <article className="rounded-[28px] border border-white/10 bg-[linear-gradient(135deg,rgba(251,113,133,0.16),rgba(245,158,11,0.08))] p-5">
                    <TrendChart
                      stroke="#fb7185"
                      fill="rgba(251,113,133,0.14)"
                      grid="rgba(255,237,213,0.14)"
                      label="rgba(255,237,213,0.7)"
                      text="rgba(255,247,237,0.86)"
                    />
                  </article>
                </div>

                <div className="grid gap-5 lg:grid-cols-[1fr_0.92fr]">
                  <article className="rounded-[28px] border border-white/10 bg-[#181018]/88 p-5">
                    <div className="flex items-end justify-between gap-3">
                      <div>
                        <p className="text-[11px] uppercase tracking-[0.24em] text-amber-200/68">
                          Heatmap
                        </p>
                        <h4 className="mt-2 text-2xl font-semibold text-white">
                          Daily momentum
                        </h4>
                      </div>
                      <span className="text-xs text-orange-50/60">
                        new AC only
                      </span>
                    </div>
                    <div className="mt-5 overflow-x-auto">
                      <Heatmap
                        shades={[
                          "#1a1217",
                          "#44203b",
                          "#8b1e3f",
                          "#d9485f",
                          "#f59e0b",
                        ]}
                        cellBorder="rgba(255, 247, 237, 0.08)"
                      />
                    </div>
                  </article>

                  <article className="rounded-[28px] border border-white/10 bg-[#181018]/88 p-5">
                    <p className="text-[11px] uppercase tracking-[0.24em] text-rose-200/68">
                      Design Read
                    </p>
                    <h4 className="mt-3 text-2xl font-semibold text-white">
                      When to use this
                    </h4>
                    <ul className="mt-4 space-y-3 text-sm leading-7 text-orange-50/76">
                      <li>适合把公开首页做得更有竞技海报感和传播感。</li>
                      <li>对大数字、冲榜氛围、成长感表达最强。</li>
                      <li>风险是信息密度再提高时，需要小心不让页面变吵。</li>
                    </ul>
                  </article>
                </div>
              </div>

              <aside className="space-y-5">
                <article className="rounded-[28px] border border-white/10 bg-[#181018]/88 p-5">
                  <div className="flex items-end justify-between gap-3">
                    <div>
                      <p className="text-[11px] uppercase tracking-[0.24em] text-amber-200/68">
                        Leaderboard
                      </p>
                      <h4 className="mt-2 text-2xl font-semibold text-white">
                        Ranking burst
                      </h4>
                    </div>
                    <span className="rounded-full border border-white/10 px-3 py-1 text-xs text-orange-50/70">
                      +/-
                    </span>
                  </div>
                  <div className="mt-5 space-y-3">
                    {leaderboardRows.map((row) => (
                      <div
                        key={row.rank}
                        className="grid grid-cols-[42px_minmax(0,1fr)_72px_56px] items-center gap-3 rounded-[18px] border border-white/8 bg-white/5 px-3 py-3"
                      >
                        <span className="text-sm text-orange-50/54">
                          {row.rank}
                        </span>
                        <span className="truncate text-sm text-white">
                          {row.user}
                        </span>
                        <span className="text-right text-sm text-orange-50/92">
                          {row.score}
                        </span>
                        <span
                          className={`text-right text-sm ${row.delta.startsWith("+") ? "text-amber-300" : "text-rose-300"}`}
                        >
                          {row.delta}
                        </span>
                      </div>
                    ))}
                  </div>
                </article>

                <article className="rounded-[28px] border border-white/10 bg-[#181018]/88 p-5">
                  <div className="flex items-end justify-between gap-3">
                    <div>
                      <p className="text-[11px] uppercase tracking-[0.24em] text-rose-200/68">
                        AC Stream
                      </p>
                      <h4 className="mt-2 text-2xl font-semibold text-white">
                        Solved set
                      </h4>
                    </div>
                    <span className="text-xs text-orange-50/60">
                      aggregated view
                    </span>
                  </div>
                  <div className="mt-5 space-y-3">
                    {problemRows.map((problem) => (
                      <div
                        key={problem.id}
                        className="rounded-[20px] border border-white/8 bg-[linear-gradient(135deg,rgba(255,255,255,0.06),rgba(255,255,255,0.02))] px-4 py-4"
                      >
                        <div className="flex flex-wrap items-start justify-between gap-3">
                          <div>
                            <p className="text-sm font-medium text-white">
                              {problem.id}
                            </p>
                            <p className="mt-1 text-xs text-orange-50/62">
                              {problem.contest}
                            </p>
                          </div>
                          <div className="text-right">
                            <p className="text-sm text-amber-200">
                              {problem.rating}
                            </p>
                            <p className="mt-1 text-xs text-orange-50/62">
                              {problem.time}
                            </p>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </article>

                <article className="rounded-[28px] border border-white/10 bg-[#181018]/88 p-5">
                  <p className="text-[11px] uppercase tracking-[0.24em] text-amber-200/68">
                    ICPC Medals
                  </p>
                  <div className="mt-4 flex flex-wrap gap-3">
                    {awardRows.map((award) => (
                      <span
                        key={award}
                        className="inline-flex rounded-full border border-white/10 bg-white/5 px-4 py-2 text-sm text-orange-50/78"
                      >
                        {award}
                      </span>
                    ))}
                  </div>
                </article>
              </aside>
            </div>
          </section>
        </div>
      </div>
    </StageShell>
  );
}

export function RootPage() {
  const [activePreviewId, setActivePreviewId] =
    useState<PreviewId>("signal-lab");
  const activePreview =
    previewOptions.find(({ id }) => id === activePreviewId) ??
    previewOptions[0];

  return (
    <main className="min-h-screen bg-[#050816] px-4 py-5 text-white sm:px-6 lg:px-8">
      <div className="mx-auto grid max-w-[1500px] gap-5 xl:grid-cols-[320px_minmax(0,1fr)]">
        <aside className="h-fit rounded-[32px] border border-white/10 bg-white/6 p-5 backdrop-blur">
          <p className="text-[11px] uppercase tracking-[0.3em] text-white/55">
            Style Sandbox
          </p>
          <h1 className="mt-4 text-3xl font-semibold tracking-tight">
            ACMRank Frontend Preview Lab
          </h1>
          <p className="mt-3 text-sm leading-7 text-white/72">
            这不是正式的 T16
            页面实现，只用于先锁定视觉语言。你选中一套后，我再把颜色和风格约束写入
            <code className="mx-1 rounded bg-white/10 px-1.5 py-0.5 text-xs">
              AGENTS.md
            </code>
            作为后续全局约束。
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
          <div className="mt-6 rounded-[24px] border border-white/10 bg-black/20 p-4">
            <p className="text-xs uppercase tracking-[0.24em] text-white/50">
              Current Read
            </p>
            <p className="mt-3 text-sm leading-6 text-white/78">
              {activePreview.recommendation}
            </p>
          </div>
        </aside>

        {activePreviewId === "signal-lab" ? <SignalLabPreview /> : null}
        {activePreviewId === "archive-ledger" ? <ArchiveLedgerPreview /> : null}
        {activePreviewId === "trackside-pulse" ? (
          <TracksidePulsePreview />
        ) : null}
      </div>
    </main>
  );
}
