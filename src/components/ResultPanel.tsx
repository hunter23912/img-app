import { useState } from 'react'

import type { DownloadFormat, DownloadStage, ImageTask, MessageTone } from '../types/image'

interface ResultPanelProps {
  image: string
  message: string
  messageTone: MessageTone
  format: DownloadFormat
  quality: number
  downloadStage: DownloadStage
  history: ImageTask[]
  hasMore: boolean
  isLoadingMore: boolean
  historySyncWarning: string
  onFormatChange: (format: DownloadFormat) => void
  onQualityChange: (quality: number) => void
  onDownload: () => void
  onSelectHistory: (url: string) => void
  onDeleteHistory: (id: string) => void
  onLoadMore: () => void
}

export function ResultPanel({
  image,
  message,
  messageTone,
  format,
  quality,
  downloadStage,
  history,
  hasMore,
  isLoadingMore,
  historySyncWarning,
  onFormatChange,
  onQualityChange,
  onDownload,
  onSelectHistory,
  onDeleteHistory,
  onLoadMore,
}: ResultPanelProps) {
  const [failedHistoryIDs, setFailedHistoryIDs] = useState<string[]>([])
  const [isHistoryCollapsed, setIsHistoryCollapsed] = useState(false)
  const isDownloading = downloadStage !== 'idle'
  const visibleHistory = history.filter((task) => task.status === 'succeeded' && Boolean(task.image))
  const displayedHistory = isHistoryCollapsed ? visibleHistory.slice(0, 5) : visibleHistory

  function handleHistoryImageError(id: string) {
    setFailedHistoryIDs((current) =>
      current.includes(id) ? current : [...current, id],
    )
  }

  function handleDeleteHistory(id: string) {
    setFailedHistoryIDs((current) => current.filter((item) => item !== id))
    onDeleteHistory(id)
  }

  const downloadStatus =
    downloadStage === 'processing'
      ? format === 'jpg'
        ? 'JPG 压缩中...'
        : '正在准备 PNG...'
      : '图片下载中...'
  const messageStyle = {
    info: 'border-sky-200/80 bg-sky-50/80 text-sky-900',
    success: 'border-emerald-200/80 bg-emerald-50/80 text-emerald-900',
    error: 'border-rose-200 bg-rose-50 text-rose-900',
  }[messageTone]

  return (
    <section className="card rounded-[1.25rem] border border-white/70 bg-white/75 shadow-[0_18px_60px_rgba(15,23,42,0.08)] backdrop-blur dark:border-slate-600/50 dark:bg-slate-800/80 dark:shadow-[0_18px_60px_rgba(0,0,0,0.32)] sm:rounded-[1.4rem]">
      <div className="card-body gap-3 p-4 sm:gap-4 sm:p-5">
        <div>
          <h2 className="card-title text-xl font-black text-slate-950">结果</h2>
          {message && (
            <div
              className={`mt-3 flex items-start gap-2.5 rounded-2xl border px-4 py-3 text-sm font-semibold leading-relaxed ${messageStyle}`}
              role={messageTone === 'error' ? 'alert' : 'status'}
              aria-live={messageTone === 'error' ? 'assertive' : 'polite'}
              aria-atomic="true"
            >
              <svg
                className="mt-0.5 h-4 w-4 shrink-0"
                viewBox="0 0 20 20"
                fill="currentColor"
                aria-hidden="true"
              >
                {messageTone === 'error' ? (
                  <path
                    fillRule="evenodd"
                    d="M10 18a8 8 0 100-16 8 8 0 000 16Zm.75-11.75a.75.75 0 00-1.5 0v4.5a.75.75 0 001.5 0v-4.5ZM10 14.5a1 1 0 100-2 1 1 0 000 2Z"
                    clipRule="evenodd"
                  />
                ) : (
                  <path
                    fillRule="evenodd"
                    d="M10 18a8 8 0 100-16 8 8 0 000 16Zm.75-11.75a.75.75 0 00-1.5 0v.5a.75.75 0 001.5 0v-.5Zm0 3a.75.75 0 00-1.5 0v4a.75.75 0 001.5 0v-4Z"
                    clipRule="evenodd"
                  />
                )}
              </svg>
              <span className="min-w-0 [overflow-wrap:anywhere]">{message}</span>
            </div>
          )}
        </div>

        <div
          className={`overflow-hidden rounded-[1.25rem] border border-white/80 bg-gradient-to-br from-slate-50 to-sky-50/70 shadow-inner shadow-slate-200/70 dark:border-slate-700 dark:from-slate-950 dark:to-slate-800/80 dark:shadow-[inset_0_2px_8px_rgba(0,0,0,0.35)] ${
            image ? 'p-2' : 'px-4 py-3'
          }`}
        >
          {image ? (
            <img
              className="mx-auto h-auto max-w-full rounded-2xl"
              src={image}
              alt="生成结果"
            />
          ) : (
            <div className="text-center text-sm font-semibold text-slate-400">
              生成后的图片会显示在这里
            </div>
          )}
        </div>

        {image && (
          <div className="grid gap-4 rounded-2xl border border-slate-200/80 bg-slate-50/70 p-4 dark:border-slate-700/80 dark:bg-slate-900/60">
            <label className="form-control grid gap-2">
              <span className="label-text font-bold text-slate-800">下载格式</span>
              <select
                className="select select-bordered h-12 w-full rounded-2xl border-slate-200 bg-white/80 shadow-inner shadow-slate-100 transition focus:border-sky-400 focus:outline-sky-200 dark:border-slate-600 dark:bg-slate-950/60 dark:shadow-[inset_0_2px_6px_rgba(0,0,0,0.3)] dark:focus:border-sky-400 dark:focus:outline-sky-900"
                value={format}
                onChange={(event) => onFormatChange(event.target.value as DownloadFormat)}
                disabled={isDownloading}
              >
                <option value="jpg">JPG 压缩图</option>
                <option value="png">PNG 原图</option>
              </select>
            </label>

            {format === 'jpg' && (
              <label className="form-control grid gap-2">
                <span className="flex items-center justify-between gap-3">
                  <span className="label-text font-bold text-slate-800">JPG 质量</span>
                  <output className="text-sm font-black tabular-nums text-sky-700" htmlFor="download-quality">
                    {quality}%
                  </output>
                </span>
                <input
                  id="download-quality"
                  className="range range-primary w-full max-w-none"
                  type="range"
                  min={1}
                  max={100}
                  step={1}
                  value={quality}
                  onChange={(event) => onQualityChange(Number(event.target.value))}
                  disabled={isDownloading}
                  aria-label="JPG 质量"
                />
                <span className="flex justify-between px-1 text-xs font-semibold text-slate-500">
                  <span>1%</span>
                  <span>100%</span>
                </span>
              </label>
            )}

            {isDownloading && (
              <div
                className="flex min-h-11 items-center gap-3 rounded-xl border border-sky-200/80 bg-sky-50/80 px-3 text-sm font-bold text-sky-800"
                role="status"
                aria-live="polite"
                aria-busy="true"
              >
                <span className="loading loading-spinner loading-sm" />
                <span>{downloadStatus}</span>
              </div>
            )}

            <button
              className="btn btn-primary min-h-12 w-full rounded-2xl font-black"
              type="button"
              onClick={onDownload}
              disabled={isDownloading}
            >
              {isDownloading
                ? downloadStage === 'processing'
                  ? '处理中...'
                  : '下载中...'
                : '下载图片'}
            </button>
          </div>
        )}

        {(visibleHistory.length > 0 || historySyncWarning) && (
          <div className="grid gap-3 border-t border-slate-200/80 pt-4">
            <div className="flex items-center justify-between gap-3">
              <h3 className="text-sm font-black text-slate-800">历史记录</h3>
              <span className="text-xs font-bold tabular-nums text-slate-500">
                {visibleHistory.length}/50
              </span>
            </div>

            {historySyncWarning && (
              <p
                className="text-sm font-semibold leading-relaxed text-amber-800 [overflow-wrap:anywhere]"
                role="alert"
              >
                {historySyncWarning}
              </p>
            )}

            {displayedHistory.length > 0 && (
              <div
                className="grid grid-cols-5 gap-2"
                aria-label="最近生成或编辑的图片"
              >
                {displayedHistory.map((task, index) => {
                  const isFailed = failedHistoryIDs.includes(task.id)
                  const isCurrent = Boolean(task.image) && image === task.image

                  return (
                    <div key={task.id} className="relative min-w-0">
                      {isFailed ? (
                        <div
                          className={`flex aspect-square items-center justify-center rounded-xl border border-rose-200 bg-rose-50 px-1.5 text-center text-[11px] font-bold leading-tight text-rose-700 ${
                            isCurrent ? 'ring-2 ring-rose-400 ring-offset-2' : ''
                          }`}
                          role="status"
                        >
                          {task.status === 'failed' ? '失败' : '图片未保存'}
                        </div>
                      ) : (
                        <button
                          className={`block aspect-square w-full overflow-hidden rounded-xl border bg-slate-100 transition hover:-translate-y-0.5 hover:shadow-md focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sky-500 ${
                            isCurrent
                              ? 'border-sky-500 ring-2 ring-sky-400 ring-offset-2'
                              : 'border-slate-200/80'
                          }`}
                          type="button"
                          aria-label={`恢复第 ${index + 1} 张历史图片`}
                          aria-pressed={isCurrent}
                          title="恢复这张历史图片"
                          onClick={() => onSelectHistory(task.image)}
                        >
                          <img
                            className="h-full w-full object-cover"
                            src={task.image}
                            alt={`历史图片 ${index + 1}`}
                            onError={() => handleHistoryImageError(task.id)}
                          />
                        </button>
                      )}
                      <button
                        className="absolute right-1 top-1 inline-flex h-7 w-7 items-center justify-center rounded-lg bg-slate-950/75 text-white shadow-sm transition hover:bg-rose-600 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sky-500"
                        type="button"
                        aria-label={`删除第 ${index + 1} 条历史记录`}
                        title="删除这条历史记录"
                        onClick={() => handleDeleteHistory(task.id)}
                      >
                        <svg
                          className="h-3.5 w-3.5"
                          viewBox="0 0 20 20"
                          fill="currentColor"
                          aria-hidden="true"
                        >
                          <path d="M6.25 4.75a.75.75 0 0 1 .75-.75h6a.75.75 0 0 1 .75.75v.5h1a.75.75 0 0 1 0 1.5h-.25v8.5A1.75 1.75 0 0 1 12.75 17h-5.5A1.75 1.75 0 0 1 5.5 15.25v-8.5h-.25a.75.75 0 0 1 0-1.5h1v-.5Zm1.5.75v.25h4.5V5.5h-4.5Zm-.75 2.75v7a.25.25 0 0 0 .25.25h5.5a.25.25 0 0 0 .25-.25v-7H7Zm1.5 1.25a.75.75 0 0 1 .75.75v3a.75.75 0 0 1-1.5 0v-3a.75.75 0 0 1 .75-.75Zm3 0a.75.75 0 0 1 .75.75v3a.75.75 0 0 1-1.5 0v-3a.75.75 0 0 1 .75-.75Z" />
                        </svg>
                      </button>
                    </div>
                  )
                })}
              </div>
            )}
            {(hasMore || visibleHistory.length > 5) && (
              <div className="grid gap-2 sm:grid-cols-2">
                {hasMore && (
                  <button
                    className="btn btn-ghost min-h-10 rounded-xl font-bold text-sky-700"
                    type="button"
                    onClick={() => {
                      setIsHistoryCollapsed(false)
                      onLoadMore()
                    }}
                    disabled={isLoadingMore}
                  >
                    {isLoadingMore ? '加载中...' : '加载更多'}
                  </button>
                )}
                {visibleHistory.length > 5 && (
                  <button
                    className="btn btn-ghost min-h-10 rounded-xl font-bold text-slate-600"
                    type="button"
                    onClick={() => setIsHistoryCollapsed((current) => !current)}
                    aria-expanded={!isHistoryCollapsed}
                  >
                    {isHistoryCollapsed ? '展开全部' : '收起'}
                  </button>
                )}
              </div>
            )}
          </div>
        )}
      </div>
    </section>
  )
}
