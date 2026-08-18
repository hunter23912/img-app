interface StatusCardProps {
  health: "checking" | "online" | "offline"
}

export function StatusCard({ health }: StatusCardProps) {
  const checking = health === "checking"
  const online = health === "online"
  const stateLabel = checking ? "检测中" : online ? "后端在线" : "后端未连接"
  const stateClass = checking ? "bg-amber-500" : online ? "bg-emerald-500" : "bg-rose-500"
  const surfaceClass = checking
    ? "border-amber-200/80 bg-amber-50/95 text-amber-900 dark:border-amber-700/80 dark:bg-amber-950/70 dark:text-amber-200"
    : online
      ? "border-emerald-200/80 bg-emerald-50/95 text-emerald-900 dark:border-emerald-700/80 dark:bg-emerald-950/70 dark:text-emerald-200"
      : "border-rose-200/80 bg-rose-50/95 text-rose-900 dark:border-rose-700/80 dark:bg-rose-950/70 dark:text-rose-200"

  return (
    <div
      className={`inline-flex min-h-9 shrink-0 items-center gap-1.5 rounded-full border px-2.5 py-1.5 text-xs font-black shadow-[0_5px_14px_rgba(15,23,42,0.08)] dark:shadow-[0_4px_14px_rgba(0,0,0,0.28)] ${surfaceClass}`}
      role="status"
      aria-live="polite"
      aria-label={`后端状态：${stateLabel}`}
    >
      <span className={`h-1.5 w-1.5 shrink-0 rounded-full ${stateClass}`} aria-hidden="true" />
      <span>{stateLabel}</span>
    </div>
  )
}
