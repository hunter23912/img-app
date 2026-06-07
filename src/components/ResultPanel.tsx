interface ResultPanelProps {
  image: string
  message: string
  isSharing: boolean
  onShare: () => void
}

export function ResultPanel({ image, message, isSharing, onShare }: ResultPanelProps) {
  return (
    <section className="card rounded-[1.4rem] border border-white/70 bg-white/75 shadow-[0_18px_60px_rgba(15,23,42,0.08)] backdrop-blur">
      <div className="card-body gap-4 p-5">
        <div>
          <h2 className="card-title text-xl font-black text-slate-950">结果</h2>
          <div className="mt-3 rounded-2xl border border-sky-200/80 bg-sky-50/80 px-4 py-3 text-sm leading-relaxed text-sky-900">
            <span>{message}</span>
          </div>
        </div>

        <div
          className={`grid overflow-hidden rounded-[1.25rem] border border-white/80 bg-gradient-to-br from-slate-50 to-sky-50/70 shadow-inner shadow-slate-200/70 ${
            image ? 'p-2' : 'min-h-64 place-items-center'
          }`}
        >
          {image ? (
            <img
              className="mx-auto h-auto max-h-[75svh] max-w-full rounded-2xl object-contain"
              src={image}
              alt="生成结果"
            />
          ) : (
            <div className="px-6 text-center text-sm font-semibold text-slate-400">
              生成后的图片会显示在这里
            </div>
          )}
        </div>

        {image && (
          <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
            <button
              className="btn btn-primary w-full rounded-2xl font-black"
              type="button"
              onClick={onShare}
              disabled={isSharing}
            >
              {isSharing ? (
                <>
                  <span className="loading loading-spinner loading-sm" />
                  准备中...
                </>
              ) : (
                '分享/保存'
              )}
            </button>
            <a
              className="btn btn-soft btn-primary w-full rounded-2xl font-black"
              href={image}
              download="gpt-image.png"
            >
              下载图片
            </a>
          </div>
        )}
      </div>
    </section>
  )
}
