import { useEffect, useRef, useState } from 'react'
import type { FormEvent } from 'react'

import { downloadImage, editImage, generateImage } from './api/images'
import { ImageFormPanel } from './components/ImageFormPanel'
import { ImageProfilesPage } from './components/ImageProfilesPage'
import { ResultPanel } from './components/ResultPanel'
import { StatusCard } from './components/StatusCard'
import { defaultAspectRatio, defaultModel, defaultResolution, getStandardImageSize, keepOriginalSize } from './constants/image'
import { useBackendHealth } from './hooks/useBackendHealth'
import { useImageHistory } from './hooks/useImageHistory'
import { useImageProfiles } from './hooks/useImageProfiles'
import { useImageModels } from './hooks/useImageModels'
import { useSourceImagePreview } from './hooks/useSourceImagePreview'
import { useTheme } from './hooks/useTheme'
import type { AppTab, DownloadFormat, DownloadStage, ImageMode, MessageTone } from './types/image'

function App() {
  const [activeTab, setActiveTab] = useState<AppTab>('image')
  const [mode, setMode] = useState<ImageMode>('generate')
  const [prompt, setPrompt] = useState('')
  const [model, setModel] = useState(defaultModel)
  const [generateAspectRatio, setGenerateAspectRatio] = useState(defaultAspectRatio)
  const [generateResolution, setGenerateResolution] = useState(defaultResolution)
  const [editAspectRatio, setEditAspectRatio] = useState(defaultAspectRatio)
  const [editResolution, setEditResolution] = useState(defaultResolution)
  const [resultImage, setResultImage] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [downloadFormat, setDownloadFormat] = useState<DownloadFormat>('jpg')
  const [downloadQuality, setDownloadQuality] = useState(95)
  const [downloadStage, setDownloadStage] = useState<DownloadStage>('idle')
  const [message, setMessage] = useState('')
  const [messageTone, setMessageTone] = useState<MessageTone>('info')
  const resultPanelRef = useRef<HTMLDivElement>(null)
  const [resultWasCleared, setResultWasCleared] = useState(false)

  const isDownloading = downloadStage !== 'idle'

  const { health } = useBackendHealth()
  const { profiles, isLoading: isProfilesLoading, isSaving: isProfilesSaving, error: profilesError, create, update, activate, remove } = useImageProfiles()
  const { theme, toggleTheme } = useTheme()
  const {
    history,
    hasMore,
    isLoadingMore,
    syncWarning: historySyncWarning,
    refreshHistory,
    loadMore,
    removeTask,
  } =
    useImageHistory()
  const { sourceImage, sourcePreview, sourceSize, selectSourceImage } = useSourceImagePreview()
  const isSettingsBusy = isProfilesLoading
  const activeProfile = profiles.find((profile) => profile.is_active)
  const { models, isLoading: isModelsLoading, isSaving: isModelsSaving, error: modelsError, add: addModel, remove: removeModel } = useImageModels(activeProfile?.id ?? '')
  const currentProfileName = isProfilesLoading ? '读取中' : activeProfile?.name || '默认配置'
  const generateSize = getStandardImageSize(generateAspectRatio, generateResolution)

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

  const latestHistoryImage = history.find((task) => task.status === 'succeeded' && Boolean(task.image))?.image ?? ''
  const currentResultImage = resultImage || (resultWasCleared ? '' : latestHistoryImage)

  function handleImageChange(file: File | null) {
    setResultWasCleared(true)
    selectSourceImage(file)
    setResultImage('')
  }

  function handleModeChange(nextMode: ImageMode) {
    setMode(nextMode)
  }

  function navigateToMode(nextMode: ImageMode) {
    handleModeChange(nextMode)
    setActiveTab('image')
  }

  function handleModelChange(nextModel: string) {
    setModel(nextModel)
  }

  const selectedModel = !isModelsLoading && models.some((option) => option.value === model)
    ? model
    : defaultModel

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

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
    setResultWasCleared(true)
    setResultImage('')
    setMessage(mode === 'generate' ? '正在请求中转站生成图片...' : '正在请求中转站编辑图片...')
    setMessageTone('info')

    try {
      const image =
        mode === 'generate'
          ? await generateImage(prompt, generateSize, selectedModel, (partialImage) => {
              setResultImage(partialImage)
              setMessage('正在接收图片预览...')
              setMessageTone('info')
            })
          : await submitEditRequest((partialImage) => {
              setResultImage(partialImage)
              setMessage('正在接收图片预览...')
              setMessageTone('info')
            })

      setResultImage(image)
      void refreshHistory()
      setMessage(mode === 'generate' ? '图片生成完成。' : '图片编辑完成。')
      setMessageTone('success')
    } catch (error) {
      void refreshHistory()
      setMessage(error instanceof Error ? error.message : '图片生成失败。')
      setMessageTone('error')
    } finally {
      setIsSubmitting(false)
    }
  }

  async function submitEditRequest(onPartialImage: (image: string) => void) {
    if (!sourceImage) {
      throw new Error('图编辑模式需要先上传一张原图。')
    }

    const size = editAspectRatio === keepOriginalSize
      ? sourceSize
      : getStandardImageSize(editAspectRatio, editResolution)
    return editImage({ prompt, size, image: sourceImage, model: selectedModel, onPartialImage })
  }

  async function handleDownloadImage() {
    if (!currentResultImage || isDownloading) return

    setDownloadStage('processing')
    setMessage('')

    try {
      const { response, filename } = await downloadImage({
        source: currentResultImage,
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

  function handleSelectHistory(url: string) {
    setResultImage(url)
    setMessage('已恢复历史图片。')
    setMessageTone('success')
  }

  return (
    <main className="mx-auto flex min-h-svh w-full max-w-xl flex-col gap-2 px-3 pb-5 pt-3 sm:gap-3 sm:px-5 sm:pb-5 sm:pt-4">
      <header className="flex flex-wrap items-center justify-between gap-2 px-1 sm:gap-3">
        <h1 className="text-3xl font-black leading-tight tracking-tight text-slate-950 dark:text-slate-100">
          像素工坊
        </h1>
        <div className="flex w-full items-center justify-end gap-1 sm:w-auto sm:gap-2">
          <nav className="flex min-w-0 items-center rounded-full border border-slate-200/80 bg-white/80 p-0.5 text-[11px] font-black shadow-sm dark:border-slate-700 dark:bg-slate-800/80 sm:text-xs" aria-label="主导航">
            <button className={`rounded-full px-2 py-1.5 transition sm:px-2.5 ${activeTab === 'image' && mode === 'generate' ? 'bg-sky-500 text-white' : 'text-slate-500 hover:text-slate-800 dark:text-slate-400 dark:hover:text-slate-100'}`} type="button" onClick={() => navigateToMode('generate')}>文生图</button>
            <button className={`rounded-full px-2 py-1.5 transition sm:px-2.5 ${activeTab === 'image' && mode === 'edit' ? 'bg-sky-500 text-white' : 'text-slate-500 hover:text-slate-800 dark:text-slate-400 dark:hover:text-slate-100'}`} type="button" onClick={() => navigateToMode('edit')}>图编辑</button>
            <button className={`rounded-full px-2 py-1.5 transition sm:px-2.5 ${activeTab === 'config' ? 'bg-sky-500 text-white' : 'text-slate-500 hover:text-slate-800 dark:text-slate-400 dark:hover:text-slate-100'}`} type="button" onClick={() => setActiveTab('config')}>配置</button>
          </nav>
          <button
            className="group inline-flex h-9 max-w-36 min-w-0 items-center gap-1.5 rounded-2xl border border-violet-200/90 bg-violet-50/90 px-2 shadow-[0_5px_14px_rgba(124,58,237,0.1)] transition hover:border-violet-300 hover:bg-violet-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-violet-400 focus-visible:ring-offset-2 dark:border-violet-700/70 dark:bg-violet-950/40 dark:hover:border-violet-600 dark:hover:bg-violet-900/50 dark:focus-visible:ring-offset-slate-950 sm:max-w-40"
            type="button"
            onClick={() => setActiveTab('config')}
            aria-label={`当前配置：${currentProfileName}`}
            title={`当前配置：${currentProfileName}`}
          >
            <span className="shrink-0 rounded-lg bg-violet-100 px-1.5 py-0.5 text-[9px] font-black leading-none text-violet-700 dark:bg-violet-900/80 dark:text-violet-200">当前</span>
            <span className="min-w-0 truncate text-[10px] font-black text-violet-950 transition group-hover:text-violet-700 dark:text-violet-100 dark:group-hover:text-violet-50 sm:text-xs">{currentProfileName}</span>
          </button>
          <button
            className="inline-flex h-9 w-9 items-center justify-center rounded-full border border-slate-200/80 bg-white/80 text-slate-700 shadow-sm transition hover:bg-slate-100 dark:border-slate-700 dark:bg-slate-800/80 dark:text-slate-200 dark:hover:bg-slate-700"
            type="button"
            onClick={toggleTheme}
            aria-label={theme === 'dark' ? '切换到浅色模式' : '切换到暗色模式'}
            title={theme === 'dark' ? '浅色模式' : '暗色模式'}
          >
            {theme === 'dark' ? '☀' : '☾'}
          </button>
          <StatusCard
            health={health}
          />
        </div>
      </header>

      {activeTab === 'config' ? (
        <ImageProfilesPage
          profiles={profiles}
          isLoading={isProfilesLoading}
          isSaving={isProfilesSaving}
          error={profilesError}
          onCreate={create}
          onUpdate={update}
          onActivate={activate}
          onDelete={remove}
        />
      ) : (
        <div className="grid gap-3 sm:gap-4">
          <ImageFormPanel
            mode={mode}
            prompt={prompt}
            model={selectedModel}
            modelOptions={models}
            isModelsLoading={isModelsLoading}
            isModelsSaving={isModelsSaving}
            modelsError={modelsError}
            generateAspectRatio={generateAspectRatio}
            generateResolution={generateResolution}
            editAspectRatio={editAspectRatio}
            editResolution={editResolution}
            sourcePreview={sourcePreview}
            isSubmitting={isSubmitting}
            isSettingsBusy={isSettingsBusy}
            onPromptChange={setPrompt}
            onModelChange={handleModelChange}
            onAddModel={async (modelName) => addModel(modelName)}
            onDeleteModel={async (id) => {
              await removeModel(id)
            }}
            onGenerateAspectRatioChange={setGenerateAspectRatio}
            onGenerateResolutionChange={setGenerateResolution}
            onEditAspectRatioChange={setEditAspectRatio}
            onEditResolutionChange={setEditResolution}
            onSourceImageChange={handleImageChange}
            onSubmit={handleSubmit}
          />

          <div ref={resultPanelRef}>
            <ResultPanel
              image={currentResultImage}
              message={message}
              messageTone={messageTone}
              format={downloadFormat}
              quality={downloadQuality}
              downloadStage={downloadStage}
              history={history}
              hasMore={hasMore}
              isLoadingMore={isLoadingMore}
              historySyncWarning={historySyncWarning}
              onFormatChange={setDownloadFormat}
              onQualityChange={setDownloadQuality}
              onDownload={handleDownloadImage}
              onSelectHistory={handleSelectHistory}
              onDeleteHistory={removeTask}
              onLoadMore={() => void loadMore()}
            />
          </div>
        </div>
      )}
    </main>
  )
}

export default App
