export function BottomNav() {
  return (
    <nav className="fixed inset-x-0 bottom-0 z-20 border-t border-slate-200/80 bg-white/90 px-4 pb-[calc(env(safe-area-inset-bottom)+0.75rem)] pt-2 shadow-[0_-12px_30px_rgba(15,23,42,0.08)] backdrop-blur">
      <div className="mx-auto flex w-full max-w-xl justify-center">
        <span className="grid h-14 content-center justify-items-center gap-0.5 rounded-2xl bg-slate-950 px-8 text-xs font-black text-white shadow-sm">
          <span className="grid size-6 place-items-center rounded-full bg-white/15 text-base leading-none text-white" aria-hidden="true">
            +
          </span>
          <span>生图</span>
        </span>
      </div>
    </nav>
  )
}
