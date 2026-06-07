import type { AppTab } from '../types/image'

interface BottomNavProps {
  activeTab: AppTab
  onTabChange: (tab: AppTab) => void
}

const navItems: Array<{ value: AppTab; label: string; mark: string }> = [
  { value: 'image', label: '生图', mark: '+' },
  { value: 'compress', label: '压缩', mark: '%' },
  { value: 'watermark', label: '去水印', mark: '×' },
]

export function BottomNav({ activeTab, onTabChange }: BottomNavProps) {
  return (
    <nav className="fixed inset-x-0 bottom-0 z-20 border-t border-slate-200/80 bg-white/90 px-4 pb-[calc(env(safe-area-inset-bottom)+0.75rem)] pt-2 shadow-[0_-12px_30px_rgba(15,23,42,0.08)] backdrop-blur">
      <div className="mx-auto grid w-full max-w-xl grid-cols-3 gap-2">
        {navItems.map((item) => {
          const isActive = item.value === activeTab

          return (
            <button
              key={item.value}
              className={`grid h-14 content-center justify-items-center gap-0.5 rounded-2xl text-xs font-black transition ${
                isActive
                  ? 'bg-slate-950 text-white shadow-sm'
                  : 'bg-slate-100/80 text-slate-500 hover:bg-slate-200/80 hover:text-slate-800'
              }`}
              type="button"
              onClick={() => onTabChange(item.value)}
            >
              <span
                className={`grid size-6 place-items-center rounded-full text-base leading-none ${
                  isActive ? 'bg-white/15 text-white' : 'bg-white text-slate-500'
                }`}
                aria-hidden="true"
              >
                {item.mark}
              </span>
              <span>{item.label}</span>
            </button>
          )
        })}
      </div>
    </nav>
  )
}
