import type { FormEvent } from "react";

import {
  getAvailableSizes,
  keepOriginalSize,
  modelOptions,
} from "../constants/image";
import type { ImageMode } from "../types/image";

interface ImageFormPanelProps {
  mode: ImageMode;
  prompt: string;
  model: string;
  imageCount: number;
  generateSize: string;
  editSize: string;
  sourcePreview: string;
  sourceSize: string;
  isSubmitting: boolean;
  onModeChange: (mode: ImageMode) => void;
  onPromptChange: (prompt: string) => void;
  onModelChange: (model: string) => void;
  onImageCountChange: (count: number) => void;
  onGenerateSizeChange: (size: string) => void;
  onEditSizeChange: (size: string) => void;
  onSourceImageChange: (file: File | null) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}

export function ImageFormPanel({
  mode,
  prompt,
  model,
  imageCount,
  generateSize,
  editSize,
  sourcePreview,
  sourceSize,
  isSubmitting,
  onModeChange,
  onPromptChange,
  onModelChange,
  onImageCountChange,
  onGenerateSizeChange,
  onEditSizeChange,
  onSourceImageChange,
  onSubmit,
}: ImageFormPanelProps) {
  const currentModel = modelOptions.find((m) => m.value === model);
  const availableSizes = getAvailableSizes(model);

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
              mode === "generate"
                ? "bg-white text-slate-950 shadow-sm"
                : "text-slate-500 hover:text-slate-800"
            }`}
            onClick={() => onModeChange("generate")}
          >
            文生图
          </button>
          <button
            type="button"
            className={`h-11 rounded-xl text-sm font-black transition ${
              mode === "edit"
                ? "bg-white text-slate-950 shadow-sm"
                : "text-slate-500 hover:text-slate-800"
            }`}
            onClick={() => onModeChange("edit")}
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
          <span className="label-text font-bold text-slate-800">模型</span>
          <select
            className="select select-bordered h-12 w-full rounded-2xl border-slate-200 bg-white/80 shadow-inner shadow-slate-100 transition focus:border-sky-400 focus:outline-sky-200"
            value={model}
            onChange={(event) => onModelChange(event.target.value)}
          >
            {modelOptions.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        </label>

        {currentModel?.supportsN && (
          <label className="form-control grid gap-2">
            <span className="label-text font-bold text-slate-800">
              出图数量
            </span>
            <input
              type="number"
              className="input input-bordered h-12 w-full rounded-2xl border-slate-200 bg-white/80 shadow-inner shadow-slate-100 transition focus:border-sky-400 focus:outline-sky-200"
              value={imageCount}
              min={1}
              max={4}
              onChange={(event) =>
                onImageCountChange(
                  Math.max(1, Math.min(4, parseInt(event.target.value) || 1)),
                )
              }
            />
          </label>
        )}

        <label className="form-control grid gap-2">
          <span className="label-text font-bold text-slate-800">尺寸</span>
          <select
            className="select select-bordered h-12 w-full rounded-2xl border-slate-200 bg-white/80 shadow-inner shadow-slate-100 transition focus:border-sky-400 focus:outline-sky-200"
            value={mode === "generate" ? generateSize : editSize}
            onChange={(event) =>
              mode === "generate"
                ? onGenerateSizeChange(event.target.value)
                : onEditSizeChange(event.target.value)
            }
          >
            {mode === "edit" && (
              <option value={keepOriginalSize}>
                {sourceSize ? `原图尺寸（${sourceSize}）` : "原图尺寸"}
              </option>
            )}
            {availableSizes.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </label>

        {mode === "edit" && (
          <div className="grid gap-2">
            <span className="label-text font-bold text-slate-800">原图</span>
            <label className="flex cursor-pointer items-center justify-center overflow-hidden rounded-3xl border border-dashed border-sky-300/70 bg-sky-50/60 text-sm font-bold text-sky-700/70 transition hover:bg-sky-50">
              <input
                accept="image/*"
                className="hidden"
                type="file"
                onChange={(event) =>
                  onSourceImageChange(event.target.files?.[0] ?? null)
                }
              />
              {sourcePreview ? (
                <img
                  className="h-auto w-full"
                  src={sourcePreview}
                  alt="待编辑原图预览"
                />
              ) : (
                <div className="flex min-h-16 items-center justify-center gap-2 px-4 py-3">
                  <svg
                    className="h-5 w-5 shrink-0"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    strokeWidth={2}
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
                    />
                  </svg>
                  <span>上传待编辑图片</span>
                </div>
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
          ) : mode === "generate" ? (
            "生成图片"
          ) : (
            "准备编辑"
          )}
        </button>
      </form>
    </section>
  );
}
