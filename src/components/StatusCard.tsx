interface StatusCardProps {
  isConfigured: boolean
}

export function StatusCard({ isConfigured }: StatusCardProps) {
  return (
    <section className="card rounded-[1.4rem] border border-white/70 bg-white/75 shadow-[0_18px_60px_rgba(15,23,42,0.08)] backdrop-blur">
      <div className="card-body gap-4 p-5">
        <div>
          <h2 className="card-title text-xl font-black text-slate-950">后端配置</h2>
          <p className="mt-1 text-sm leading-relaxed text-slate-500">
            Endpoint 和 API key 由 Go 后端通过环境变量读取，前端不再展示密钥。
          </p>
        </div>

        <div
          className={`rounded-2xl border px-4 py-3 text-sm font-semibold ${
            isConfigured
              ? 'border-emerald-200 bg-emerald-50/80 text-emerald-800'
              : 'border-amber-200 bg-amber-50/80 text-amber-800'
          }`}
        >
          {isConfigured
            ? 'IMG_API_KEY 已配置，可以生成图片。'
            : '未检测到 IMG_API_KEY，请设置环境变量后重启后端。'}
        </div>
      </div>
    </section>
  )
}
