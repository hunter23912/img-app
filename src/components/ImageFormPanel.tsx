import { useRef, useState } from "react";
import type { FormEvent } from "react";

import {
  aspectRatioOptions,
  getResolutionOptions,
  keepOriginalSize,
} from "../constants/image";
import type { ImageMode, ModelOption, PromptApplyMode } from "../types/image";
import {
  applyPromptPreset,
  loadPromptApplyMode,
  persistPromptApplyMode,
} from "../utils/promptPresets";
import { PromptPresetPanel } from "./PromptPresetPanel";
import { ModelManagerPanel } from "./ModelManagerPanel";

interface ImageFormPanelProps {
  mode: ImageMode;
  prompt: string;
  model: string;
  modelOptions: ModelOption[];
  isModelsLoading: boolean;
  isModelsSaving: boolean;
  modelsError: string;
  generateAspectRatio: string;
  generateResolution: string;
  editAspectRatio: string;
  editResolution: string;
  sourcePreview: string;
  isSubmitting: boolean;
  isSettingsBusy: boolean;
  onPromptChange: (prompt: string) => void;
  onModelChange: (model: string) => void;
  onAddModel: (model: string) => Promise<ModelOption>;
  onDeleteModel: (id: string) => Promise<void>;
  onGenerateAspectRatioChange: (aspectRatio: string) => void;
  onGenerateResolutionChange: (resolution: string) => void;
  onEditAspectRatioChange: (aspectRatio: string) => void;
  onEditResolutionChange: (resolution: string) => void;
  onSourceImageChange: (file: File | null) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}

export function ImageFormPanel({
  mode,
  prompt,
  model,
  modelOptions,
  isModelsLoading,
  isModelsSaving,
  modelsError,
  generateAspectRatio,
  generateResolution,
  editAspectRatio,
  editResolution,
  sourcePreview,
  isSubmitting,
  isSettingsBusy,
  onPromptChange,
  onModelChange,
  onAddModel,
  onDeleteModel,
  onGenerateAspectRatioChange,
  onGenerateResolutionChange,
  onEditAspectRatioChange,
  onEditResolutionChange,
  onSourceImageChange,
  onSubmit,
}: ImageFormPanelProps) {
  const promptInputRef = useRef<HTMLTextAreaElement>(null);
  const [promptApplyMode, setPromptApplyMode] =
    useState<PromptApplyMode>(loadPromptApplyMode);
  const [isManagingModels, setIsManagingModels] = useState(false);

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
      className="btn min-h-12 w-full rounded-2xl border-0 bg-gradient-to-r from-sky-500 via-blue-500 to-indigo-500 text-base font-black text-white shadow-[0_14px_30px_rgba(59,130,246,0.28)] transition hover:scale-[1.01] hover:brightness-105 disabled:scale-100 disabled:bg-slate-300 dark:shadow-[0_14px_30px_rgba(0,0,0,0.35)] sm:min-h-13"
      type="submit"
      disabled={isSubmitting || isSettingsBusy || (mode === "edit" && !sourcePreview)}
    >
      {isSettingsBusy ? (
        <>
          <span className="loading loading-spinner loading-sm" />
          准备中...
        </>
      ) : isSubmitting ? (
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
  const currentAspectRatio = mode === "generate" ? generateAspectRatio : editAspectRatio;
  const currentResolution = mode === "generate" ? generateResolution : editResolution;
  const currentResolutionOptions = getResolutionOptions(currentAspectRatio);
  const currentSizeOptions = mode === "edit"
    ? [{ value: keepOriginalSize, label: "原图" }, ...aspectRatioOptions]
    : aspectRatioOptions;
  const currentSizeValue = currentAspectRatio;
  const currentResolutionValue = currentResolution;

  return (
    <section className="card rounded-[1.25rem] border border-white/70 bg-white/75 shadow-[0_18px_60px_rgba(15,23,42,0.08)] backdrop-blur dark:border-slate-600/50 dark:bg-slate-800/80 dark:shadow-[0_18px_60px_rgba(0,0,0,0.32)] sm:rounded-[1.4rem]">
      <form className="card-body gap-3 px-4 pb-4 pt-2 sm:gap-4 sm:px-5 sm:pb-5 sm:pt-3" onSubmit={onSubmit}>
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
                className="inline-flex min-h-10 shrink-0 items-center justify-center rounded-xl px-2.5 text-xs font-bold text-slate-500 transition hover:bg-slate-100 hover:text-slate-800 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sky-500 dark:text-slate-400 dark:hover:bg-slate-700 dark:hover:text-slate-100"
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
            className="textarea textarea-bordered min-h-28 w-full resize-y rounded-2xl border-slate-200 bg-white/80 leading-relaxed shadow-inner shadow-slate-100 transition focus:border-sky-400 focus:outline-sky-200 dark:border-slate-600 dark:bg-slate-950/60 dark:shadow-[inset_0_2px_6px_rgba(0,0,0,0.3)] dark:focus:border-sky-400 dark:focus:outline-sky-900 sm:min-h-36"
            value={prompt}
            onChange={(event) => onPromptChange(event.target.value)}
            placeholder="描述你想生成或编辑的画面"
            rows={4}
          />
        </label>

        <div className="grid gap-3 sm:gap-4">
          <div className="form-control grid min-w-0 gap-1.5 sm:gap-2">
            <div className="flex items-center justify-between gap-3">
              <span className="label-text font-bold text-slate-800">模型</span>
              <button
                className="btn btn-ghost btn-sm shrink-0 rounded-xl px-3 font-bold text-sky-700 dark:text-sky-300"
                type="button"
                aria-expanded={isManagingModels}
                onClick={() => setIsManagingModels((current) => !current)}
                disabled={isModelsLoading || isModelsSaving}
              >
                {isManagingModels ? "完成" : "管理"}
              </button>
            </div>
            <select
              className="select select-bordered h-11 min-w-0 w-full truncate rounded-2xl border-slate-200 bg-white/80 px-3 text-sm shadow-inner shadow-slate-100 transition focus:border-sky-400 focus:outline-sky-200 dark:border-slate-600 dark:bg-slate-950/60 dark:shadow-[inset_0_2px_6px_rgba(0,0,0,0.3)] dark:focus:border-sky-400 dark:focus:outline-sky-900 sm:h-12 sm:px-4 sm:text-base"
              value={model}
              onChange={(event) => onModelChange(event.target.value)}
              disabled={isModelsLoading}
            >
              {modelOptions.map((opt) => (
                <option key={opt.id} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
            {modelsError && !isManagingModels && (
              <div className="rounded-xl border border-amber-200 bg-amber-50 px-3 py-2 text-xs font-bold text-amber-700 dark:border-amber-900/70 dark:bg-amber-950/30 dark:text-amber-300" role="status">
                模型列表暂时无法加载，当前仍可使用固定模型。
              </div>
            )}
            {isManagingModels && (
              <ModelManagerPanel
                models={modelOptions}
                isSaving={isModelsSaving}
                error={modelsError}
                onAdd={onAddModel}
                onDelete={onDeleteModel}
                onAdded={onModelChange}
              />
            )}
          </div>

          <div className="grid grid-cols-2 gap-2.5 sm:gap-3">
            <label className="form-control grid min-w-0 gap-1.5 sm:gap-2">
              <span className="label-text font-bold text-slate-800">尺寸</span>
              <select
                className="select select-bordered h-11 min-w-0 w-full truncate rounded-2xl border-slate-200 bg-white/80 px-2 text-sm shadow-inner shadow-slate-100 transition focus:border-sky-400 focus:outline-sky-200 dark:border-slate-600 dark:bg-slate-950/60 dark:shadow-[inset_0_2px_6px_rgba(0,0,0,0.3)] dark:focus:border-sky-400 dark:focus:outline-sky-900 sm:h-12 sm:px-3 sm:text-base"
                value={currentSizeValue}
                onChange={(event) => {
                  if (mode === "generate") {
                    onGenerateAspectRatioChange(event.target.value);
                  } else {
                    onEditAspectRatioChange(event.target.value);
                  }
                }}
              >
                {currentSizeOptions.map((option) => (
                  <option key={option.value} value={option.value}>{option.label}</option>
                ))}
              </select>
            </label>

            <label className="form-control grid min-w-0 gap-1.5 sm:gap-2">
              <span className="label-text font-bold text-slate-800">分辨率</span>
              <select
                className="select select-bordered h-11 min-w-0 w-full truncate rounded-2xl border-slate-200 bg-white/80 px-2 text-sm shadow-inner shadow-slate-100 transition focus:border-sky-400 focus:outline-sky-200 dark:border-slate-600 dark:bg-slate-950/60 dark:shadow-[inset_0_2px_6px_rgba(0,0,0,0.3)] dark:focus:border-sky-400 dark:focus:outline-sky-900 sm:h-12 sm:px-3 sm:text-base"
                value={currentResolutionValue}
                onChange={(event) => {
                  if (mode === "generate") {
                    onGenerateResolutionChange(event.target.value);
                  } else {
                    onEditResolutionChange(event.target.value);
                  }
                }}
              >
                {currentResolutionOptions.map((option) => (
                  <option key={option.value} value={option.value}>{option.label}</option>
                ))}
              </select>
            </label>
          </div>
        </div>

        {mode === "edit" && (
          <div className="grid gap-2">
            <span className="label-text font-bold text-slate-800">原图</span>
            <label className="flex min-h-20 max-h-[min(75svh,42rem)] cursor-pointer items-center justify-center overflow-hidden rounded-3xl border border-dashed border-sky-300/70 bg-sky-50/60 text-sm font-bold text-sky-700/80 transition hover:bg-sky-50 dark:border-sky-700/70 dark:bg-slate-950/50 dark:text-sky-300 dark:hover:bg-slate-900">
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
