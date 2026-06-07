import { useState } from 'react'
import type { FormEvent } from 'react'

import { editImage, generateImage } from './api/images'
import { BottomNav } from './components/BottomNav'
import { ImageCompressionPanel } from './components/ImageCompressionPanel'
import { ImageFormPanel } from './components/ImageFormPanel'
import { ResultPanel } from './components/ResultPanel'
import { StatusCard } from './components/StatusCard'
import { WatermarkPanel } from './components/WatermarkPanel'
import { keepOriginalSize, sizeOptions } from './constants/image'
import { useBackendHealth } from './hooks/useBackendHealth'
import { useImageShare } from './hooks/useImageShare'
import { useSourceImagePreview } from './hooks/useSourceImagePreview'
import type { AppTab, ImageMode } from './types/image'

const tabTitles: Record<AppTab, { title: string; subtitle: string }> = {
  image: { title: '图片生成', subtitle: 'Generate' },
  compress: { title: '图片压缩', subtitle: 'Compress' },
  watermark: { title: '去水印', subtitle: 'Cleanup' },
}

function App() {
  const [activeTab, setActiveTab] = useState<AppTab>('image')
  const [mode, setMode] = useState<ImageMode>('generate')
  const [prompt, setPrompt] = useState('')
  const [generateSize, setGenerateSize] = useState(sizeOptions[0].value)
  const [editSize, setEditSize] = useState(sizeOptions[0].value)
  const [resultImage, setResultImage] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [message, setMessage] = useState('后端启动后会从环境变量读取中转站配置。')

  const { healthLabel, healthClass, isConfigured } = useBackendHealth()
  const { sourceImage, sourcePreview, sourceSize, selectSourceImage } = useSourceImagePreview()
  const { isSharing, shareImage } = useImageShare(setMessage)

  function handleImageChange(file: File | null) {
    selectSourceImage(file)
    setResultImage('')
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
      const image =
        mode === 'generate'
          ? await generateImage(prompt, generateSize)
          : await submitEditRequest()

      setResultImage(image)
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

    const size = editSize === keepOriginalSize ? sourceSize : editSize
    return editImage({ prompt, size, image: sourceImage })
  }

  async function handleShareImage() {
    await shareImage(resultImage)
  }

  const currentTitle = tabTitles[activeTab]
  const showBackendStatus = activeTab === 'image'

  return (
    <main className="mx-auto flex min-h-svh w-full max-w-xl flex-col gap-4 px-4 pb-28 pt-5 sm:px-5">
      <header className="flex items-start justify-between gap-4 px-1 pt-1">
        <div>
          <p className="text-xs font-bold uppercase tracking-[0.18em] text-sky-700/70">
            {currentTitle.subtitle}
          </p>
          <h1 className="mt-2 text-4xl font-black leading-none tracking-tight text-slate-950">
            {currentTitle.title}
          </h1>
        </div>
        {showBackendStatus && (
          <span className={`badge badge-soft mt-1 shrink-0 rounded-full border-0 px-3 py-3 shadow-sm ${healthClass}`}>
            {healthLabel}
          </span>
        )}
      </header>

      <div className="grid gap-3">
        {activeTab === 'image' && (
          <>
            <StatusCard isConfigured={isConfigured} />

            <ImageFormPanel
              mode={mode}
              prompt={prompt}
              generateSize={generateSize}
              editSize={editSize}
              sourcePreview={sourcePreview}
              sourceSize={sourceSize}
              isSubmitting={isSubmitting}
              onModeChange={setMode}
              onPromptChange={setPrompt}
              onGenerateSizeChange={setGenerateSize}
              onEditSizeChange={setEditSize}
              onSourceImageChange={handleImageChange}
              onSubmit={handleSubmit}
            />

            <ResultPanel
              image={resultImage}
              message={message}
              isSharing={isSharing}
              onShare={handleShareImage}
            />
          </>
        )}

        {activeTab === 'compress' && <ImageCompressionPanel />}

        {activeTab === 'watermark' && <WatermarkPanel />}
      </div>

      <BottomNav activeTab={activeTab} onTabChange={setActiveTab} />
    </main>
  )
}

export default App
