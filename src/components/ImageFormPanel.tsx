import type { FormEvent } from 'react'

import { keepOriginalSize, sizeOptions } from '../constants/image'
import type { ImageMode } from '../types/image'

interface ImageFormPanelProps {
  mode: ImageMode
  prompt: string
  generateSize: string
  editSize: string
  sourcePreview: string
  sourceSize: string
  isSubmitting: boolean
  onModeChange: (mode: ImageMode) => void
  onPromptChange: (prompt: string) => void
  onGenerateSizeChange: (size: string) => void
  onEditSizeChange: (size: string) => void
  onSourceImageChange: (file: File | null) => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
}

export function ImageFormPanel({
  mode,
  prompt,
  generateSize,
  editSize,
  sourcePreview,
  sourceSize,
  isSubmitting,
  onModeChange,
  onPromptChange,
  onGenerateSizeChange,
  onEditSizeChange,
  onSourceImageChange,
  onSubmit,
}: ImageFormPanelProps) {
  return (
    <section className="card rounded-[1.4rem] border border-white/70 bg-white/75 shadow-[0_18px_60px_rgba(15,23,42,0.08)] backdrop-blur">
      <form className="card-body gap-4 p-5" onSubmit={onSubmit}>
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
            onClick={() => onModeChange('generate')}
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
            onClick={() => onModeChange('edit')}
          >
            图编辑
          </button>
        </div>

        <label className="form-control grid gap-2">
          <span className="label-text font-bold text-slate-800">Prompt</span>
          <textarea
            className="textarea textarea-bordered min-h-36 w-full resize-y rounded-2xl border-slate-200 bg-white/80 leading-relaxed shadow-inner shadow-slate-100 transition focus:border-sky-400 focus:outline-sky-200"
            value={prompt}
            onChange={(event) => onPromptChange(event.target.value)}
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
                ? onGenerateSizeChange(event.target.value)
                : onEditSizeChange(event.target.value)
            }
          >
            {mode === 'edit' && (
              <option value={keepOriginalSize}>
                {sourceSize ? `保持原图尺寸（${sourceSize}，可能更贵）` : '保持原图尺寸（可能更贵）'}
              </option>
            )}
            {sizeOptions.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </label>

        {mode === 'edit' && (
          <div className="grid gap-2">
            <span className="label-text font-bold text-slate-800">原图</span>
            <label className="flex min-h-48 cursor-pointer items-center justify-center overflow-hidden rounded-3xl border border-dashed border-sky-300/70 bg-sky-50/60 text-sm font-bold text-sky-700/70 transition hover:bg-sky-50">
              <input
                accept="image/*"
                className="hidden"
                type="file"
                onChange={(event) => onSourceImageChange(event.target.files?.[0] ?? null)}
              />
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
      </form>
    </section>
  )
}
