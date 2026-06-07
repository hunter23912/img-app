import { useEffect, useMemo, useState } from 'react'
import type { ChangeEvent, FormEvent } from 'react'

type Mode = 'generate' | 'edit'

type HealthState = 'checking' | 'online' | 'offline'

const sizeOptions = ['1024x1024', '1024x1536', '1536x1024', '2048x2048', '2048x1152', '1152x2048']
const keepOriginalSize = 'original'

function App() {
  const [mode, setMode] = useState<Mode>('generate')
  const [prompt, setPrompt] = useState('')
  const [generateSize, setGenerateSize] = useState(sizeOptions[0])
  const [editSize, setEditSize] = useState(keepOriginalSize)
  const [sourceImage, setSourceImage] = useState<File | null>(null)
  const [sourcePreview, setSourcePreview] = useState('')
  const [sourceSize, setSourceSize] = useState('')
  const [resultImage, setResultImage] = useState('')
  const [health, setHealth] = useState<HealthState>('checking')
  const [isConfigured, setIsConfigured] = useState(false)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [message, setMessage] = useState('后端启动后会从环境变量读取中转站配置。')

  useEffect(() => {
    let ignore = false

    async function checkBackend() {
      try {
        const response = await fetch('/api/health')
        const data = (await response.json()) as { ok?: boolean; configured?: boolean }

        if (!ignore) {
          setHealth(data.ok ? 'online' : 'offline')
          setIsConfigured(Boolean(data.configured))
        }
      } catch {
        if (!ignore) {
          setHealth('offline')
          setIsConfigured(false)
        }
      }
    }

    checkBackend()

    return () => {
      ignore = true
    }
  }, [])

  useEffect(() => {
    return () => {
      if (sourcePreview) {
        URL.revokeObjectURL(sourcePreview)
      }
    }
  }, [sourcePreview])

  const healthLabel = useMemo(() => {
    if (health === 'checking') return '检测中'
    if (health === 'online') return '后端在线'
    return '后端未连接'
  }, [health])

  const healthClass = useMemo(() => {
    if (health === 'checking') return 'badge-warning'
    if (health === 'online') return 'badge-success'
    return 'badge-error'
  }, [health])

  function handleImageChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0] ?? null
    setSourceImage(file)
    setResultImage('')
    setSourceSize('')

    if (!file) {
      setSourcePreview('')
      return
    }

    if (sourcePreview) {
      URL.revokeObjectURL(sourcePreview)
    }

    const previewURL = URL.createObjectURL(file)
    setSourcePreview(previewURL)

    const image = new Image()
    image.onload = () => {
      setSourceSize(`${image.naturalWidth}x${image.naturalHeight}`)
    }
    image.src = previewURL
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    if (!isConfigured) {
      setMessage('后端还没有配置 IMG_API_KEY，请先设置环境变量并重启 Go 服务。')
      return
    }

    if (!prompt.trim()) {
      setMessage('请先填写 prompt。')
      return
    }

    if (mode === 'edit' && !sourceImage) {
      setMessage('图编辑模式需要先上传一张原图。')
      return
    }

    setIsSubmitting(true)
    setResultImage('')
    setMessage(mode === 'generate' ? '正在请求中转站生成图片...' : '正在请求中转站编辑图片...')

    try {
      const response =
        mode === 'generate'
          ? await fetch('/api/generate', {
              method: 'POST',
                headers: {
                  'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                  model: 'gpt-image-2',
                  prompt,
                  size: generateSize,
                  quality: 'auto',
                }),
              })
          : await submitEditRequest()

      const data = (await response.json()) as { image?: string; error?: string }

      if (!response.ok) {
        throw new Error(data.error || `请求失败：${response.status}`)
      }

      if (!data.image) {
        throw new Error('后端没有返回图片数据。')
      }

      setResultImage(data.image)
      setMessage(mode === 'generate' ? '图片生成完成。' : '图片编辑完成。')
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '图片生成失败。')
    } finally {
      setIsSubmitting(false)
    }
  }

  async function submitEditRequest() {
    if (!sourceImage) {
      throw new Error('图编辑模式需要先上传一张原图。')
    }

    const formData = new FormData()
    formData.append('model', 'gpt-image-2')
    formData.append('prompt', prompt)
    formData.append('quality', 'auto')
    formData.append('image', sourceImage)

    const size = editSize === keepOriginalSize ? sourceSize : editSize
    if (size) {
      formData.append('size', size)
    }

    return fetch('/api/edit', {
      method: 'POST',
      body: formData,
    })
  }

  return (
    <main className="mx-auto flex min-h-svh w-full max-w-xl flex-col gap-4 px-4 py-5 sm:px-5">
      <header className="flex items-start justify-between gap-4 px-1 pt-1">
        <div>
          <p className="text-xs font-bold uppercase tracking-[0.18em] text-sky-700/70">
            GPT Image Tool
          </p>
          <h1 className="mt-2 text-4xl font-black leading-none tracking-tight text-slate-950">
            极简图片生成
          </h1>
        </div>
        <span className={`badge badge-soft mt-1 shrink-0 rounded-full border-0 px-3 py-3 shadow-sm ${healthClass}`}>
          {healthLabel}
        </span>
      </header>

      <form className="grid gap-3" onSubmit={handleSubmit}>
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

        <section className="card rounded-[1.4rem] border border-white/70 bg-white/75 shadow-[0_18px_60px_rgba(15,23,42,0.08)] backdrop-blur">
          <div className="card-body gap-4 p-5">
            <div
              className="grid grid-cols-2 rounded-2xl bg-slate-100/80 p-1"
              role="tablist"
              aria-label="图片模式"
            >
              <button
                type="button"
                className={`h-11 rounded-xl text-sm font-black transition ${
                  mode === 'generate'
                    ? 'bg-white text-slate-950 shadow-sm'
                    : 'text-slate-500 hover:text-slate-800'
                }`}
                onClick={() => setMode('generate')}
              >
                文生图
              </button>
              <button
                type="button"
                className={`h-11 rounded-xl text-sm font-black transition ${
                  mode === 'edit'
                    ? 'bg-white text-slate-950 shadow-sm'
                    : 'text-slate-500 hover:text-slate-800'
                }`}
                onClick={() => setMode('edit')}
              >
                图编辑
              </button>
            </div>

            <label className="form-control grid gap-2">
              <span className="label-text font-bold text-slate-800">Prompt</span>
              <textarea
                className="textarea textarea-bordered min-h-36 w-full resize-y rounded-2xl border-slate-200 bg-white/80 leading-relaxed shadow-inner shadow-slate-100 transition focus:border-sky-400 focus:outline-sky-200"
                value={prompt}
                onChange={(event) => setPrompt(event.target.value)}
                placeholder="描述你想生成或编辑的画面"
                rows={5}
              />
            </label>

            <label className="form-control grid gap-2">
              <span className="label-text font-bold text-slate-800">尺寸</span>
              <select
                className="select select-bordered h-12 w-full rounded-2xl border-slate-200 bg-white/80 shadow-inner shadow-slate-100 transition focus:border-sky-400 focus:outline-sky-200"
                value={mode === 'generate' ? generateSize : editSize}
                onChange={(event) =>
                  mode === 'generate'
                    ? setGenerateSize(event.target.value)
                    : setEditSize(event.target.value)
                }
              >
                {mode === 'edit' && (
                  <option value={keepOriginalSize}>
                    {sourceSize ? `保持原图尺寸（${sourceSize}）` : '保持原图尺寸'}
                  </option>
                )}
                {sizeOptions.map((option) => (
                  <option key={option} value={option}>
                    {option}
                  </option>
                ))}
              </select>
            </label>

            {mode === 'edit' && (
              <div className="grid gap-2">
                <span className="label-text font-bold text-slate-800">原图</span>
                <label className="flex min-h-48 cursor-pointer items-center justify-center overflow-hidden rounded-3xl border border-dashed border-sky-300/70 bg-sky-50/60 text-sm font-bold text-sky-700/70 transition hover:bg-sky-50">
                  <input accept="image/*" className="hidden" type="file" onChange={handleImageChange} />
                  {sourcePreview ? (
                    <img
                      className="h-full max-h-72 w-full object-contain"
                      src={sourcePreview}
                      alt="待编辑原图预览"
                    />
                  ) : (
                    <span>上传待编辑图片</span>
                  )}
                </label>
                {sourceSize && (
                  <p className="px-1 text-xs font-semibold text-slate-500">
                    当前原图尺寸：{sourceSize}
                  </p>
                )}
              </div>
            )}

            <button
              className="btn min-h-13 w-full rounded-2xl border-0 bg-gradient-to-r from-sky-500 via-blue-500 to-indigo-500 text-base font-black text-white shadow-[0_14px_30px_rgba(59,130,246,0.28)] transition hover:scale-[1.01] hover:brightness-105 disabled:scale-100 disabled:bg-slate-300"
              type="submit"
              disabled={isSubmitting}
            >
              {isSubmitting ? (
                <>
                  <span className="loading loading-spinner loading-sm" />
                  生成中...
                </>
              ) : mode === 'generate' ? (
                '生成图片'
              ) : (
                '准备编辑'
              )}
            </button>
          </div>
        </section>

        <section className="card rounded-[1.4rem] border border-white/70 bg-white/75 shadow-[0_18px_60px_rgba(15,23,42,0.08)] backdrop-blur">
          <div className="card-body gap-4 p-5">
            <div>
              <h2 className="card-title text-xl font-black text-slate-950">结果</h2>
              <div className="mt-3 rounded-2xl border border-sky-200/80 bg-sky-50/80 px-4 py-3 text-sm leading-relaxed text-sky-900">
                <span>{message}</span>
              </div>
            </div>

            <div className="grid aspect-square place-items-center overflow-hidden rounded-[1.25rem] border border-white/80 bg-gradient-to-br from-slate-50 to-sky-50/70 shadow-inner shadow-slate-200/70">
              {resultImage ? (
                <img className="h-full w-full object-contain" src={resultImage} alt="生成结果" />
              ) : (
                <div className="px-6 text-center text-sm font-semibold text-slate-400">
                  生成后的图片会显示在这里
                </div>
              )}
            </div>

            {resultImage && (
              <a
                className="btn btn-soft btn-primary w-full rounded-2xl font-black"
                href={resultImage}
                download="gpt-image.png"
              >
                下载图片
              </a>
            )}
          </div>
        </section>
      </form>
    </main>
  )
}

export default App
