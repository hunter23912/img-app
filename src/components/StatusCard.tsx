interface StatusCardProps {
  isConfigured: boolean;
  health: "checking" | "online" | "offline";
}

export function StatusCard({ isConfigured, health }: StatusCardProps) {
  const backendReady = health === "online";
  const allReady = backendReady && isConfigured;
  const checking = health === "checking";
  const stateLabel = checking
    ? "检测中"
    : allReady
      ? "已就绪"
      : backendReady
        ? "Key 未配置"
        : "后端未连接";
  const stateClass = allReady
    ? "bg-emerald-500"
    : checking || backendReady
      ? "bg-amber-500"
      : "bg-rose-500";
  const surfaceClass = allReady
    ? "border-emerald-200/80 bg-emerald-50/95 text-emerald-900 dark:border-emerald-700/80 dark:bg-emerald-950/70 dark:text-emerald-200"
    : checking || backendReady
      ? "border-amber-200/80 bg-amber-50/95 text-amber-900 dark:border-amber-700/80 dark:bg-amber-950/70 dark:text-amber-200"
      : "border-rose-200/80 bg-rose-50/95 text-rose-900 dark:border-rose-700/80 dark:bg-rose-950/70 dark:text-rose-200";

  const backendStatus = checking ? "检测中" : backendReady ? "在线" : "未连接";
  const keyStatus = checking ? "检测中" : isConfigured ? "已配置" : "未配置";
  const backendDotClass = checking
    ? "bg-amber-500"
    : backendReady
      ? "bg-sky-500"
      : "bg-rose-500";
  const keyDotClass = checking
    ? "bg-amber-500"
    : isConfigured
      ? "bg-violet-500"
      : "bg-rose-500";

  return (
    <details className="group relative shrink-0">
      <summary
        className={`flex min-h-9 cursor-pointer list-none items-center gap-1.5 rounded-full border px-2.5 py-1.5 text-xs font-black shadow-[0_5px_14px_rgba(15,23,42,0.08)] outline-none transition hover:brightness-[0.98] focus-visible:ring-2 focus-visible:ring-sky-400 focus-visible:ring-offset-2 dark:focus-visible:ring-offset-slate-950 dark:shadow-[0_4px_14px_rgba(0,0,0,0.28)] [&::-webkit-details-marker]:hidden ${surfaceClass}`}
        aria-label={`服务状态：${stateLabel}`}
      >
        <span
          className={`h-1.5 w-1.5 shrink-0 rounded-full ${stateClass}`}
          aria-hidden="true"
        />
        <span>{stateLabel}</span>
        <svg
          className="h-3 w-3 opacity-60 transition-transform group-open:rotate-180"
          viewBox="0 0 20 20"
          fill="currentColor"
          aria-hidden="true"
        >
          <path
            fillRule="evenodd"
            d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06Z"
            clipRule="evenodd"
          />
        </svg>
      </summary>

      <div
        className="absolute right-0 top-[calc(100%+0.5rem)] z-30 w-56 max-w-[calc(100vw-1.5rem)] rounded-2xl border border-slate-200/80 bg-white/95 p-2 shadow-[0_14px_32px_rgba(15,23,42,0.14)] backdrop-blur dark:border-slate-600/70 dark:bg-slate-900/95 dark:shadow-[0_16px_36px_rgba(0,0,0,0.42)]"
      >
        <div className="flex items-center gap-2 rounded-xl border border-sky-100 bg-sky-50/80 px-2.5 py-2 dark:border-sky-800/80 dark:bg-slate-800/90">
          <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-md bg-sky-100 text-[11px] font-black text-sky-700 dark:bg-sky-950 dark:text-sky-300">
            1
          </span>
          <span className="min-w-0 flex-1">
            <span className="block text-xs font-black text-slate-800 dark:text-slate-100">后端连接</span>
            <span className="block text-[11px] font-semibold text-slate-500 dark:text-slate-400">{backendStatus}</span>
          </span>
          <span className={`h-2 w-2 shrink-0 rounded-full ${backendDotClass}`} aria-hidden="true" />
        </div>
        <div className="mt-1.5 flex items-center gap-2 rounded-xl border border-violet-100 bg-violet-50/80 px-2.5 py-2 dark:border-violet-800/80 dark:bg-slate-800/90">
          <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-md bg-violet-100 text-[11px] font-black text-violet-700 dark:bg-violet-950 dark:text-violet-300">
            2
          </span>
          <span className="min-w-0 flex-1">
            <span className="block text-xs font-black text-slate-800 dark:text-slate-100">API Key</span>
            <span className="block text-[11px] font-semibold text-slate-500 dark:text-slate-400">{keyStatus}</span>
          </span>
          <span className={`h-2 w-2 shrink-0 rounded-full ${keyDotClass}`} aria-hidden="true" />
        </div>
      </div>
    </details>
  );
}
