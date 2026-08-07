import { useRef, useState } from "react";
import type { FormEvent } from "react";

import {
  keepOriginalSize,
  modelOptions,
  sizeOptions,
} from "../constants/image";
import type { ImageMode, PromptApplyMode } from "../types/image";
import {
  applyPromptPreset,
  loadPromptApplyMode,
  persistPromptApplyMode,
} from "../utils/promptPresets";
import { PromptPresetPanel } from "./PromptPresetPanel";

interface ImageFormPanelProps {
  mode: ImageMode;
  prompt: string;
  model: string;
  generateSize: string;
  editSize: string;
  sourcePreview: string;
  isSubmitting: boolean;
  onModeChange: (mode: ImageMode) => void;
  onPromptChange: (prompt: string) => void;
  onModelChange: (model: string) => void;
  onGenerateSizeChange: (size: string) => void;
  onEditSizeChange: (size: string) => void;
  onSourceImageChange: (file: File | null) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}

export function ImageFormPanel({
  mode,
  prompt,
  model,
  generateSize,
  editSize,
  sourcePreview,
  isSubmitting,
  onModeChange,
  onPromptChange,
  onModelChange,
  onGenerateSizeChange,
  onEditSizeChange,
  onSourceImageChange,
  onSubmit,
}: ImageFormPanelProps) {
  const promptInputRef = useRef<HTMLTextAreaElement>(null);
  const [promptApplyMode, setPromptApplyMode] =
    useState<PromptApplyMode>(loadPromptApplyMode);

  function handleApplyModeChange(nextMode: PromptApplyMode) {
    setPromptApplyMode(nextMode);
    persistPromptApplyMode(nextMode);
  }

  function handleApplyPrompt(value: string) {
    onPromptChange(applyPromptPreset(prompt, value, promptApplyMode));
    window.requestAnimationFrame(() => promptInputRef.current?.focus());
  }

  const submitButton = (
    <button
      className="btn min-h-12 w-full rounded-2xl border-0 bg-gradient-to-r from-sky-500 via-blue-500 to-indigo-500 text-base font-black text-white shadow-[0_14px_30px_rgba(59,130,246,0.28)] transition hover:scale-[1.01] hover:brightness-105 disabled:scale-100 disabled:bg-slate-300 sm:min-h-13"
      type="submit"
      disabled={isSubmitting || (mode === "edit" && !sourcePreview)}
    >
      {isSubmitting ? (
        <>
          <span className="loading loading-spinner loading-sm" />
          {mode === "generate" ? "生成中..." : "编辑中..."}
        </>
      ) : mode === "generate" ? (
        "生成图片"
      ) : (
        "开始编辑"
      )}
    </button>
  );

  return (
    <section className="card rounded-[1.25rem] border border-white/70 bg-white/75 shadow-[0_18px_60px_rgba(15,23,42,0.08)] backdrop-blur sm:rounded-[1.4rem]">
      <form className="card-body gap-3 p-4 sm:gap-4 sm:p-5" onSubmit={onSubmit}>
        <div
          className="grid grid-cols-2 rounded-xl bg-slate-100/80 p-0.5"
          role="tablist"
          aria-label="图片模式"
        >
          <button
            type="button"
            className={`h-10 rounded-lg text-sm font-black transition ${
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
            className={`h-10 rounded-lg text-sm font-black transition ${
              mode === "edit"
                ? "bg-white text-slate-950 shadow-sm"
                : "text-slate-500 hover:text-slate-800"
            }`}
            onClick={() => onModeChange("edit")}
          >
            图编辑
          </button>
        </div>

        <PromptPresetPanel
          mode={mode}
          applyMode={promptApplyMode}
          onApplyModeChange={handleApplyModeChange}
          onApplyPrompt={handleApplyPrompt}
        />

        <label className="form-control grid gap-1.5 sm:gap-2">
          <span className="flex items-center justify-between gap-3">
            <span className="label-text font-bold text-slate-800">Prompt</span>
            {prompt.trim() && (
              <button
                className="inline-flex min-h-10 shrink-0 items-center justify-center rounded-xl px-2.5 text-xs font-bold text-slate-500 transition hover:bg-slate-100 hover:text-slate-800 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sky-500"
                type="button"
                onClick={() => {
                  onPromptChange("");
                  window.requestAnimationFrame(() => promptInputRef.current?.focus());
                }}
              >
                清空
              </button>
            )}
          </span>
          <textarea
            ref={promptInputRef}
            className="textarea textarea-bordered min-h-28 w-full resize-y rounded-2xl border-slate-200 bg-white/80 leading-relaxed shadow-inner shadow-slate-100 transition focus:border-sky-400 focus:outline-sky-200 sm:min-h-36"
            value={prompt}
            onChange={(event) => onPromptChange(event.target.value)}
            placeholder="描述你想生成或编辑的画面"
            rows={4}
          />
        </label>

        <div className="grid grid-cols-2 gap-2.5 max-[359px]:grid-cols-1 sm:gap-3">
          <label className="form-control grid min-w-0 gap-1.5 sm:gap-2">
            <span className="label-text font-bold text-slate-800">模型</span>
            <select
              className="select select-bordered h-11 min-w-0 w-full truncate rounded-2xl border-slate-200 bg-white/80 px-3 text-sm shadow-inner shadow-slate-100 transition focus:border-sky-400 focus:outline-sky-200 sm:h-12 sm:px-4 sm:text-base"
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

          <label className="form-control grid min-w-0 gap-1.5 sm:gap-2">
            <span className="label-text font-bold text-slate-800">尺寸</span>
            <select
              className="select select-bordered h-11 min-w-0 w-full truncate rounded-2xl border-slate-200 bg-white/80 px-3 text-sm shadow-inner shadow-slate-100 transition focus:border-sky-400 focus:outline-sky-200 sm:h-12 sm:px-4 sm:text-base"
              value={mode === "generate" ? generateSize : editSize}
              onChange={(event) =>
                mode === "generate"
                  ? onGenerateSizeChange(event.target.value)
                  : onEditSizeChange(event.target.value)
              }
            >
              {mode === "edit" && (
                <option value={keepOriginalSize}>原图</option>
              )}
              {sizeOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </label>
        </div>

        {mode === "edit" && (
          <div className="grid gap-2">
            <span className="label-text font-bold text-slate-800">原图</span>
            <label className="flex min-h-20 max-h-[min(75svh,42rem)] cursor-pointer items-center justify-center overflow-hidden rounded-3xl border border-dashed border-sky-300/70 bg-sky-50/60 text-sm font-bold text-sky-700/80 transition hover:bg-sky-50">
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
                  className="max-h-[min(75svh,42rem)] w-full object-contain"
                  src={sourcePreview}
                  alt="待编辑原图预览，点击图片可更换"
                />
              ) : (
                <span className="flex min-h-20 items-center justify-center gap-2 px-4 py-3">
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
                  <span>上传图片</span>
                </span>
              )}
            </label>
            {submitButton}
          </div>
        )}

        {mode === "generate" && submitButton}
      </form>
    </section>
  );
}
