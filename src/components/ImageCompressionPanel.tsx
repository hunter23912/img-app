import { useEffect, useState } from 'react'

import { compressImageOnServer } from '../api/images'
import {
  compressionPercent,
  extensionFromOutput,
  formatBytes,
  type CompressionOutputType,
} from '../utils/compressImage'

const outputOptions: Array<{ value: CompressionOutputType; label: string }> = [
  { value: 'auto', label: '自动' },
  { value: 'image/jpeg', label: 'JPG' },
  { value: 'image/png', label: 'PNG' },
]

interface CompressionResult {
  url: string
  blob: Blob
  width: number
  height: number
  sourceFormat: string
  output: string
  originalBytes: number
  compressedBytes: number
  savedBytes: number
}

export function ImageCompressionPanel() {
  const [file, setFile] = useState<File | null>(null)
  const [sourcePreview, setSourcePreview] = useState('')
  const [result, setResult] = useState<CompressionResult | null>(null)
  const [quality, setQuality] = useState(0.82)
  const [outputType, setOutputType] = useState<CompressionOutputType>('auto')
  const [isCompressing, setIsCompressing] = useState(false)
  const [message, setMessage] = useState('等待选择图片。')

  useEffect(() => {
    return () => {
      if (sourcePreview) URL.revokeObjectURL(sourcePreview)
    }
  }, [sourcePreview])

  useEffect(() => {
    return () => {
      if (result?.url) URL.revokeObjectURL(result.url)
    }
  }, [result])

  function handleFileChange(nextFile: File | null) {
    setFile(nextFile)
    setResult(null)

    if (!nextFile) {
      setSourcePreview('')
      setMessage('等待选择图片。')
      return
    }

    setSourcePreview(URL.createObjectURL(nextFile))
    setMessage(`原图大小：${formatBytes(nextFile.size)}`)
  }

  async function handleCompress() {
    if (!file) {
      setMessage('请先选择一张图片。')
      return
    }

    setIsCompressing(true)
    setResult(null)
    setMessage('正在压缩图片...')

    try {
      const compressed = await compressImageOnServer({
        image: file,
        quality,
        outputType,
      })

      setResult(compressed)
      const percent = compressionPercent(compressed.originalBytes, compressed.compressedBytes)
      setMessage(
        compressed.savedBytes > 0
          ? `压缩完成：节省 ${formatBytes(compressed.savedBytes)}，约 ${percent}%。`
          : `已完成重编码，但结果比原图大 ${formatBytes(Math.abs(compressed.savedBytes))}。建议改用 JPG 或降低质量。`
      )
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '图片压缩失败。')
    } finally {
      setIsCompressing(false)
    }
  }

  const downloadName = `compressed-image.${extensionFromOutput(result?.output || 'jpg')}`
  const savedPercent = result ? compressionPercent(result.originalBytes, result.compressedBytes) : 0

  return (
    <section className="card rounded-[1.4rem] border border-white/70 bg-white/75 shadow-[0_18px_60px_rgba(15,23,42,0.08)] backdrop-blur">
      <div className="card-body gap-4 p-5">
        <div className="flex items-center justify-between gap-3">
          <h2 className="card-title text-xl font-black text-slate-950">图片压缩</h2>
          <span className="badge badge-soft badge-info rounded-full border-0 px-3 py-3">工具</span>
        </div>

        <div className="grid gap-3">
          <label className="flex min-h-44 cursor-pointer items-center justify-center overflow-hidden rounded-3xl border border-dashed border-cyan-300/70 bg-cyan-50/60 text-sm font-bold text-cyan-700/70 transition hover:bg-cyan-50">
            <input
              accept="image/*"
              className="hidden"
              type="file"
              onChange={(event) => handleFileChange(event.target.files?.[0] ?? null)}
            />
            {sourcePreview ? (
              <img
                className="h-full max-h-64 w-full object-contain"
                src={sourcePreview}
                alt="待压缩图片预览"
              />
            ) : (
              <span>选择图片</span>
            )}
          </label>

          <div className="grid grid-cols-1 gap-3 sm:grid-cols-[1fr_1.4fr]">
            <label className="form-control grid gap-2">
              <span className="label-text font-bold text-slate-800">格式</span>
              <select
                className="select select-bordered h-12 w-full rounded-2xl border-slate-200 bg-white/80 shadow-inner shadow-slate-100 transition focus:border-cyan-400 focus:outline-cyan-200"
                value={outputType}
                onChange={(event) => setOutputType(event.target.value as CompressionOutputType)}
              >
                {outputOptions.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </label>

            <div className="grid content-end">
              <div className="rounded-2xl border border-slate-200 bg-white/70 px-4 py-3 text-sm font-semibold text-slate-600">
                保持原图尺寸，仅压缩编码
              </div>
            </div>
          </div>

          {outputType !== 'image/png' && (
            <label className="form-control grid gap-2">
              <span className="label-text font-bold text-slate-800">
                JPG 质量 {Math.round(quality * 100)}%
              </span>
              <input
                className="range range-info"
                max="0.95"
                min="0.45"
                step="0.01"
                type="range"
                value={quality}
                onChange={(event) => setQuality(Number(event.target.value))}
              />
            </label>
          )}

          <button
            className="btn min-h-13 w-full rounded-2xl border-0 bg-gradient-to-r from-cyan-500 via-sky-500 to-blue-500 text-base font-black text-white shadow-[0_14px_30px_rgba(14,165,233,0.24)] transition hover:scale-[1.01] hover:brightness-105 disabled:scale-100 disabled:bg-slate-300"
            type="button"
            disabled={isCompressing}
            onClick={handleCompress}
          >
            {isCompressing ? (
              <>
                <span className="loading loading-spinner loading-sm" />
                压缩中...
              </>
            ) : (
              '压缩图片'
            )}
          </button>
        </div>

        <div className="rounded-2xl border border-cyan-200/80 bg-cyan-50/80 px-4 py-3 text-sm leading-relaxed text-cyan-900">
          {message}
        </div>

        {result && (
          <div className="grid gap-3">
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
              <Metric label="原图" value={formatBytes(result.originalBytes)} />
              <Metric label="输出" value={formatBytes(result.compressedBytes)} />
              <Metric
                label={result.savedBytes >= 0 ? '节省' : '增大'}
                value={formatBytes(Math.abs(result.savedBytes))}
                tone={result.savedBytes >= 0 ? 'good' : 'warn'}
              />
              <Metric
                label="压缩率"
                value={result.savedBytes >= 0 ? `${savedPercent}%` : '0%'}
                tone={result.savedBytes >= 0 ? 'good' : 'warn'}
              />
            </div>

            <div className="grid overflow-hidden rounded-[1.25rem] border border-white/80 bg-gradient-to-br from-slate-50 to-cyan-50/70 p-2 shadow-inner shadow-slate-200/70">
              <img
                className="mx-auto h-auto max-h-[55svh] max-w-full rounded-2xl object-contain"
                src={result.url}
                alt="压缩结果"
              />
            </div>

            <div className="grid grid-cols-1 gap-2 sm:grid-cols-[1fr_auto] sm:items-center">
              <p className="px-1 text-xs font-semibold text-slate-500">
                输出尺寸：{result.width}x{result.height}（保持原尺寸） · 输出格式：{result.output.toUpperCase()}
              </p>
              <a
                className="btn btn-soft btn-info w-full rounded-2xl font-black sm:w-auto"
                href={result.url}
                download={downloadName}
              >
                下载压缩图
              </a>
            </div>
          </div>
        )}
      </div>
    </section>
  )
}

function Metric({
  label,
  value,
  tone = 'default',
}: {
  label: string
  value: string
  tone?: 'default' | 'good' | 'warn'
}) {
  const toneClass =
    tone === 'good'
      ? 'border-emerald-200 bg-emerald-50/80 text-emerald-800'
      : tone === 'warn'
        ? 'border-amber-200 bg-amber-50/80 text-amber-800'
        : 'border-slate-200 bg-white/70 text-slate-700'

  return (
    <div className={`rounded-2xl border px-3 py-2 ${toneClass}`}>
      <div className="text-[0.68rem] font-bold text-current/65">{label}</div>
      <div className="mt-1 text-sm font-black">{value}</div>
    </div>
  )
}
