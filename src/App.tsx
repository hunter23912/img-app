import { useEffect, useRef, useState } from 'react'
import type { FormEvent } from 'react'

import { downloadImage, editImage, generateImage } from './api/images'
import { ImageFormPanel } from './components/ImageFormPanel'
import { ResultPanel } from './components/ResultPanel'
import { StatusCard } from './components/StatusCard'
import { defaultModel, defaultSize, keepOriginalSize } from './constants/image'
import { useBackendHealth } from './hooks/useBackendHealth'
import { useSourceImagePreview } from './hooks/useSourceImagePreview'
import type { DownloadFormat, DownloadStage, ImageMode, MessageTone } from './types/image'

function App() {
  const [mode, setMode] = useState<ImageMode>('generate')
  const [prompt, setPrompt] = useState('')
  const [model, setModel] = useState(defaultModel)
  const [generateSize, setGenerateSize] = useState(defaultSize)
  const [editSize, setEditSize] = useState(defaultSize)
  const [resultImage, setResultImage] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [downloadFormat, setDownloadFormat] = useState<DownloadFormat>('jpg')
  const [downloadQuality, setDownloadQuality] = useState(95)
  const [downloadStage, setDownloadStage] = useState<DownloadStage>('idle')
  const [message, setMessage] = useState('')
  const [messageTone, setMessageTone] = useState<MessageTone>('info')
  const resultPanelRef = useRef<HTMLDivElement>(null)

  const isDownloading = downloadStage !== 'idle'

  const { health, isConfigured } = useBackendHealth()
  const { sourceImage, sourcePreview, sourceSize, selectSourceImage } = useSourceImagePreview()

  useEffect(() => {
    if (!message || messageTone !== 'error') return

    const panel = resultPanelRef.current
    if (!panel) return

    const bounds = panel.getBoundingClientRect()
    if (bounds.top < 0 || bounds.top > window.innerHeight) {
      const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
      panel.scrollIntoView({ behavior: reduceMotion ? 'auto' : 'smooth', block: 'nearest' })
    }
  }, [message, messageTone])

  function handleImageChange(file: File | null) {
    selectSourceImage(file)
    setResultImage('')
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    if (!isConfigured) {
      setMessage('后端还没有配置 IMG_API_KEY，请先设置环境变量并重启 Go 服务。')
      setMessageTone('error')
      return
    }

    if (!prompt.trim()) {
      setMessage('请先填写 prompt。')
      setMessageTone('error')
      return
    }

    if (mode === 'edit' && !sourceImage) {
      setMessage('图编辑模式需要先上传一张原图。')
      setMessageTone('error')
      return
    }

    setIsSubmitting(true)
    setResultImage('')
    setMessage(mode === 'generate' ? '正在请求中转站生成图片...' : '正在请求中转站编辑图片...')
    setMessageTone('info')

    try {
        const image =
        mode === 'generate'
          ? await generateImage(prompt, generateSize, model)
          : await submitEditRequest()

      setResultImage(image)
      setMessage(mode === 'generate' ? '图片生成完成。' : '图片编辑完成。')
      setMessageTone('success')
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '图片生成失败。')
      setMessageTone('error')
    } finally {
      setIsSubmitting(false)
    }
  }

  async function submitEditRequest() {
    if (!sourceImage) {
      throw new Error('图编辑模式需要先上传一张原图。')
    }

    const size = editSize === keepOriginalSize ? sourceSize : editSize
    return editImage({ prompt, size, image: sourceImage, model })
  }

  async function handleDownloadImage() {
    if (!resultImage || isDownloading) return

    setDownloadStage('processing')
    setMessage('')

    try {
      const { response, filename } = await downloadImage({
        source: resultImage,
        format: downloadFormat,
        quality: downloadQuality,
      })
      setDownloadStage('downloading')
      const blob = await response.blob()
      const objectURL = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = objectURL
      link.download = filename
      document.body.appendChild(link)
      link.click()
      link.remove()
      window.setTimeout(() => URL.revokeObjectURL(objectURL), 1000)
      setMessage('图片下载已开始。')
      setMessageTone('success')
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '图片下载失败。')
      setMessageTone('error')
    } finally {
      setDownloadStage('idle')
    }
  }

  return (
    <main className="mx-auto flex min-h-svh w-full max-w-xl flex-col gap-2 px-3 pb-5 pt-3 sm:gap-3 sm:px-5 sm:pb-5 sm:pt-4">
      <header className="flex items-center justify-between gap-2 px-1 sm:gap-3">
        <h1 className="text-3xl font-black leading-tight tracking-tight text-slate-950">
          像素工坊
        </h1>
        <StatusCard
          isConfigured={isConfigured}
          health={health}
        />
      </header>

      <div className="grid gap-3 sm:gap-4">
        <ImageFormPanel
          mode={mode}
          prompt={prompt}
          model={model}
          generateSize={generateSize}
          editSize={editSize}
          sourcePreview={sourcePreview}
          isSubmitting={isSubmitting}
          onModeChange={setMode}
          onPromptChange={setPrompt}
          onModelChange={setModel}
          onGenerateSizeChange={setGenerateSize}
          onEditSizeChange={setEditSize}
          onSourceImageChange={handleImageChange}
          onSubmit={handleSubmit}
        />

        <div ref={resultPanelRef}>
          <ResultPanel
            image={resultImage}
            message={message}
            messageTone={messageTone}
            format={downloadFormat}
            quality={downloadQuality}
            downloadStage={downloadStage}
            onFormatChange={setDownloadFormat}
            onQualityChange={setDownloadQuality}
            onDownload={handleDownloadImage}
          />
        </div>
      </div>
    </main>
  )
}

export default App
