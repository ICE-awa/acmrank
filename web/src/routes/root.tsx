const highlights = [
  "公开用户页展示 Codeforces / AtCoder / 洛谷 聚合 AC 结果",
  "支持平台聚合视图、单账号视图和 ICPC 奖项历史",
  "个人页保留 SCNU Rating 折线图与每日新 AC 热力图入口",
] as const;

export function RootPage() {
  return (
    <main className="min-h-screen bg-[radial-gradient(circle_at_top,#1d4ed8_0%,#0f172a_45%,#020617_100%)] px-6 py-16 text-slate-100">
      <div className="mx-auto flex max-w-5xl flex-col gap-10">
        <section className="space-y-5">
          <span className="inline-flex rounded-full border border-sky-300/40 bg-sky-400/10 px-3 py-1 text-sm tracking-[0.2em] text-sky-100 uppercase">
            Round 1 Bootstrap
          </span>
          <div className="space-y-4">
            <h1 className="text-5xl font-semibold tracking-tight">ACMRank</h1>
            <p className="max-w-3xl text-lg leading-8 text-slate-200">
              华南师范大学校内竞赛档案与训练排行榜系统，公开展示聚合 AC 题目、
              SCNU Rating 走势、平台视图与 ICPC 奖项历史。
            </p>
          </div>
        </section>

        <section className="grid gap-4 md:grid-cols-3">
          {highlights.map((highlight) => (
            <article
              key={highlight}
              className="rounded-3xl border border-white/10 bg-slate-950/35 p-5 shadow-[0_24px_80px_rgba(2,6,23,0.35)] backdrop-blur"
            >
              <p className="text-sm leading-7 text-slate-100">{highlight}</p>
            </article>
          ))}
        </section>
      </div>
    </main>
  );
}
