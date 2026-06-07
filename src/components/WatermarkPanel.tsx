import { useEffect, useRef, useState } from 'react'

import { removeWatermark } from '../api/images'
import { formatBytes } from '../utils/compressImage'

interface WatermarkResult {
  url: string
  blob: Blob
  mode: string
}

export function WatermarkPanel() {
  const imageCanvasRef = useRef<HTMLCanvasElement | null>(null)
  const maskCanvasRef = useRef<HTMLCanvasElement | null>(null)
  const [file, setFile] = useState<File | null>(null)
  const [sourceURL, setSourceURL] = useState('')
  const [result, setResult] = useState<WatermarkResult | null>(null)
  const [brushSize, setBrushSize] = useState(34)
  const [isDrawing, setIsDrawing] = useState(false)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [hasMask, setHasMask] = useState(false)
  const [message, setMessage] = useState('选择图片后，在水印区域涂抹标记。')

  useEffect(() => {
    return () => {
      if (sourceURL) URL.revokeObjectURL(sourceURL)
    }
  }, [sourceURL])

  useEffect(() => {
    return () => {
      if (result?.url) URL.revokeObjectURL(result.url)
    }
  }, [result])

  function handleFileChange(nextFile: File | null) {
    setFile(nextFile)
    setResult(null)
    setHasMask(false)

    if (!nextFile) {
      setSourceURL('')
      clearCanvas()
      setMessage('选择图片后，在水印区域涂抹标记。')
      return
    }

    const nextURL = URL.createObjectURL(nextFile)
    setSourceURL(nextURL)
    setMessage(`原图大小：${formatBytes(nextFile.size)}。请涂抹水印区域。`)

    const image = new Image()
    image.onload = () => {
      drawSourceImage(image)
    }
    image.src = nextURL
  }

  function drawSourceImage(image: HTMLImageElement) {
    const imageCanvas = imageCanvasRef.current
    const maskCanvas = maskCanvasRef.current
    if (!imageCanvas || !maskCanvas) return

    imageCanvas.width = image.naturalWidth
    imageCanvas.height = image.naturalHeight
    maskCanvas.width = image.naturalWidth
    maskCanvas.height = image.naturalHeight

    const imageContext = imageCanvas.getContext('2d')
    const maskContext = maskCanvas.getContext('2d')
    if (!imageContext || !maskContext) return

    imageContext.clearRect(0, 0, imageCanvas.width, imageCanvas.height)
    imageContext.drawImage(image, 0, 0)

    maskContext.clearRect(0, 0, maskCanvas.width, maskCanvas.height)
  }

  function clearMask() {
    const maskCanvas = maskCanvasRef.current
    const maskContext = maskCanvas?.getContext('2d')
    if (!maskCanvas || !maskContext) return

    maskContext.clearRect(0, 0, maskCanvas.width, maskCanvas.height)
    setHasMask(false)
    setMessage(file ? '已清空标记区域。' : '选择图片后，在水印区域涂抹标记。')
  }

  function clearCanvas() {
    const imageCanvas = imageCanvasRef.current
    const maskCanvas = maskCanvasRef.current
    imageCanvas?.getContext('2d')?.clearRect(0, 0, imageCanvas.width, imageCanvas.height)
    maskCanvas?.getContext('2d')?.clearRect(0, 0, maskCanvas.width, maskCanvas.height)
  }

  function beginDraw(event: React.PointerEvent<HTMLCanvasElement>) {
    if (!file) return
    setIsDrawing(true)
    drawPoint(event)
  }

  function drawMove(event: React.PointerEvent<HTMLCanvasElement>) {
    if (!isDrawing) return
    drawPoint(event)
  }

  function endDraw() {
    setIsDrawing(false)
  }

  function drawPoint(event: React.PointerEvent<HTMLCanvasElement>) {
    const canvas = maskCanvasRef.current
    const context = canvas?.getContext('2d')
    if (!canvas || !context) return

    const rect = canvas.getBoundingClientRect()
    const scaleX = canvas.width / rect.width
    const scaleY = canvas.height / rect.height
    const x = (event.clientX - rect.left) * scaleX
    const y = (event.clientY - rect.top) * scaleY
    const radius = (brushSize / 2) * Math.max(scaleX, scaleY)

    context.fillStyle = 'rgba(255,255,255,0.82)'
    context.beginPath()
    context.arc(x, y, radius, 0, Math.PI * 2)
    context.fill()
    setHasMask(true)
  }

  async function handleSubmit() {
    if (!file) {
      setMessage('请先选择一张图片。')
      return
    }
    if (!hasMask) {
      setMessage('请先涂抹要处理的水印区域。')
      return
    }

    const maskCanvas = maskCanvasRef.current
    if (!maskCanvas) return

    setIsSubmitting(true)
    setResult(null)
    setMessage('正在提交去水印任务...')

    try {
      const mask = await canvasToPNG(maskCanvas)
      const nextResult = await removeWatermark({ image: file, mask })
      setResult(nextResult)
      setMessage(
        nextResult.mode === 'placeholder'
          ? '最小流程已跑通：后端收到图片和标记区域。当前先返回原图占位。'
          : '去水印处理完成。'
      )
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '去水印失败。')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <section className="card rounded-[1.4rem] border border-white/70 bg-white/75 shadow-[0_18px_60px_rgba(15,23,42,0.08)] backdrop-blur">
      <div className="card-body gap-4 p-5">
        <div className="flex items-center justify-between gap-3">
          <h2 className="card-title text-xl font-black text-slate-950">去水印</h2>
          <span className="badge badge-soft badge-warning rounded-full border-0 px-3 py-3">标记模式</span>
        </div>

        <label className="flex min-h-40 cursor-pointer items-center justify-center rounded-3xl border border-dashed border-amber-300/70 bg-amber-50/70 px-6 text-center text-sm font-bold text-amber-800 transition hover:bg-amber-50">
          <input
            accept="image/*"
            className="hidden"
            type="file"
            onChange={(event) => handleFileChange(event.target.files?.[0] ?? null)}
          />
          {file ? file.name : '选择待处理图片'}
        </label>

        <div className="grid gap-2">
          <div className="relative overflow-hidden rounded-[1.25rem] border border-white/80 bg-slate-100 shadow-inner shadow-slate-200/70">
            <canvas className="block h-auto max-h-[58svh] w-full touch-none object-contain" ref={imageCanvasRef} />
            <canvas
              className="absolute inset-0 h-full w-full touch-none opacity-70"
              ref={maskCanvasRef}
              onPointerDown={beginDraw}
              onPointerLeave={endDraw}
              onPointerMove={drawMove}
              onPointerUp={endDraw}
            />
            {!file && (
              <div className="absolute inset-0 grid place-items-center px-6 text-center text-sm font-semibold text-slate-400">
                图片和标记区域会显示在这里
              </div>
            )}
          </div>

          <label className="form-control grid gap-2">
            <span className="label-text font-bold text-slate-800">画笔 {brushSize}px</span>
            <input
              className="range range-warning"
              max="90"
              min="12"
              step="2"
              type="range"
              value={brushSize}
              onChange={(event) => setBrushSize(Number(event.target.value))}
            />
          </label>
        </div>

        <div className="grid grid-cols-2 gap-2">
          <button
            className="btn btn-soft btn-warning rounded-2xl font-black"
            type="button"
            onClick={clearMask}
          >
            清空标记
          </button>
          <button
            className="btn rounded-2xl border-0 bg-gradient-to-r from-amber-500 via-orange-500 to-rose-500 font-black text-white shadow-[0_14px_30px_rgba(245,158,11,0.22)]"
            type="button"
            disabled={isSubmitting}
            onClick={handleSubmit}
          >
            {isSubmitting ? (
              <>
                <span className="loading loading-spinner loading-sm" />
                处理中...
              </>
            ) : (
              '提交处理'
            )}
          </button>
        </div>

        <div className="rounded-2xl border border-amber-200/80 bg-amber-50/80 px-4 py-3 text-sm leading-relaxed text-amber-900">
          {message}
        </div>

        {result && (
          <div className="grid gap-3">
            <div className="grid overflow-hidden rounded-[1.25rem] border border-white/80 bg-gradient-to-br from-slate-50 to-amber-50/70 p-2 shadow-inner shadow-slate-200/70">
              <img
                className="mx-auto h-auto max-h-[55svh] max-w-full rounded-2xl object-contain"
                src={result.url}
                alt="去水印结果"
              />
            </div>
            <a
              className="btn btn-soft btn-warning w-full rounded-2xl font-black"
              href={result.url}
              download="watermark-result.png"
            >
              下载结果
            </a>
          </div>
        )}
      </div>
    </section>
  )
}

function canvasToPNG(canvas: HTMLCanvasElement) {
  return new Promise<Blob>((resolve, reject) => {
    canvas.toBlob((blob) => {
      if (!blob) {
        reject(new Error('标记区域生成失败。'))
        return
      }
      resolve(blob)
    }, 'image/png')
  })
}
