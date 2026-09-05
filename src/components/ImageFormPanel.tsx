import { useRef, useState } from "react";
import type { FormEvent } from "react";

import { aspectRatioOptions, getResolutionOptions, keepOriginalSize } from "../constants/image";
import { maxSourceImages } from "../hooks/useSourceImagePreview";
import type { ImageMode, ModelOption, PromptApplyMode, SourceImage } from "../types/image";
import { applyPromptPreset, loadPromptApplyMode, persistPromptApplyMode } from "../utils/promptPresets";
import { findImageMention } from "../utils/imageReferences";
import { ModelManagerPanel } from "./ModelManagerPanel";
import { PromptPresetPanel } from "./PromptPresetPanel";

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
  sourceImages: SourceImage[];
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
  onMainImageChange: (file: File | null) => void;
  onReferenceImagesChange: (files: File[]) => void;
  onRemoveSourceImage: (id: string) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}

export function ImageFormPanel({
  mode, prompt, model, modelOptions, isModelsLoading, isModelsSaving, modelsError,
  generateAspectRatio, generateResolution, editAspectRatio, editResolution, sourceImages,
  isSubmitting, isSettingsBusy, onPromptChange, onModelChange, onAddModel, onDeleteModel,
  onGenerateAspectRatioChange, onGenerateResolutionChange, onEditAspectRatioChange,
  onEditResolutionChange, onMainImageChange, onReferenceImagesChange, onRemoveSourceImage,
  onSubmit,
}: ImageFormPanelProps) {
  const promptInputRef = useRef<HTMLTextAreaElement>(null);
  const [promptApplyMode, setPromptApplyMode] = useState<PromptApplyMode>(loadPromptApplyMode);
  const [isManagingModels, setIsManagingModels] = useState(false);
  const [imageMention, setImageMention] = useState<{ start: number; end: number; query: string } | null>(null);
  const [activeMentionIndex, setActiveMentionIndex] = useState(0);

  function updateImageMention(value: string, cursor: number) {
    setImageMention(findImageMention(value, cursor));
    setActiveMentionIndex(0);
  }

  function insertImageMention(imageNumber: number) {
    if (!imageMention) return;
    const value = `${prompt.slice(0, imageMention.start)}@${imageNumber} ${prompt.slice(imageMention.end)}`;
    const nextCursor = imageMention.start + String(imageNumber).length + 2;
    onPromptChange(value);
    setImageMention(null);
    window.requestAnimationFrame(() => {
      promptInputRef.current?.focus();
      promptInputRef.current?.setSelectionRange(nextCursor, nextCursor);
    });
  }

  const mentionOptions = imageMention
    ? sourceImages.map((image, index) => ({ image, number: index + 1 })).filter(({ number }) => String(number).startsWith(imageMention.query))
    : [];
  const currentAspectRatio = mode === "generate" ? generateAspectRatio : editAspectRatio;
  const currentResolution = mode === "generate" ? generateResolution : editResolution;
  const currentResolutionOptions = getResolutionOptions(currentAspectRatio);
  const currentSizeOptions = mode === "edit" ? [{ value: keepOriginalSize, label: "原图" }, ...aspectRatioOptions] : aspectRatioOptions;
  const submitButton = (
    <button className="btn min-h-12 w-full rounded-2xl border-0 bg-gradient-to-r from-sky-500 via-blue-500 to-indigo-500 text-base font-black text-white shadow-[0_14px_30px_rgba(59,130,246,0.28)] transition hover:scale-[1.01] hover:brightness-105 disabled:scale-100 disabled:bg-slate-300 dark:shadow-[0_14px_30px_rgba(0,0,0,0.35)] sm:min-h-13" type="submit" disabled={isSubmitting || isSettingsBusy || (mode === "edit" && sourceImages.length === 0)}>
      {isSettingsBusy ? <><span className="loading loading-spinner loading-sm" />准备中...</> : isSubmitting ? <><span className="loading loading-spinner loading-sm" />{mode === "generate" ? "生成中..." : "编辑中..."}</> : mode === "generate" ? "生成图片" : "开始编辑"}
    </button>
  );

  return (
    <section className="card rounded-[1.25rem] border border-white/70 bg-white/75 shadow-[0_18px_60px_rgba(15,23,42,0.08)] backdrop-blur dark:border-slate-600/50 dark:bg-slate-800/80 dark:shadow-[0_18px_60px_rgba(0,0,0,0.32)] sm:rounded-[1.4rem]">
      <form className="card-body gap-3 px-4 pb-4 pt-2 sm:gap-4 sm:px-5 sm:pb-5 sm:pt-3" onSubmit={onSubmit}>
        <PromptPresetPanel mode={mode} applyMode={promptApplyMode} onApplyModeChange={(nextMode) => { setPromptApplyMode(nextMode); persistPromptApplyMode(nextMode); }} onApplyPrompt={(value) => { onPromptChange(applyPromptPreset(prompt, value, promptApplyMode)); window.requestAnimationFrame(() => promptInputRef.current?.focus()); }} />
        <label className="form-control grid gap-1.5 sm:gap-2">
          <span className="flex items-center justify-between gap-3"><span className="label-text font-bold text-slate-800">Prompt</span>{prompt.trim() && <button className="inline-flex min-h-10 shrink-0 items-center justify-center rounded-xl px-2.5 text-xs font-bold text-slate-500 transition hover:bg-slate-100 hover:text-slate-800 dark:text-slate-400 dark:hover:bg-slate-700 dark:hover:text-slate-100" type="button" onClick={() => { onPromptChange(""); window.requestAnimationFrame(() => promptInputRef.current?.focus()); }}>清空</button>}</span>
          <div className="relative">
            <textarea ref={promptInputRef} className="textarea textarea-bordered min-h-28 w-full resize-y rounded-2xl border-slate-200 bg-white/80 leading-relaxed shadow-inner shadow-slate-100 transition focus:border-sky-400 focus:outline-sky-200 dark:border-slate-600 dark:bg-slate-950/60 dark:shadow-[inset_0_2px_6px_rgba(0,0,0,0.3)] dark:focus:border-sky-400 dark:focus:outline-sky-900 sm:min-h-36" value={prompt} onChange={(event) => { onPromptChange(event.target.value); updateImageMention(event.target.value, event.target.selectionStart); }} onClick={(event) => updateImageMention(event.currentTarget.value, event.currentTarget.selectionStart)} onKeyUp={(event) => updateImageMention(event.currentTarget.value, event.currentTarget.selectionStart)} onBlur={() => setImageMention(null)} onKeyDown={(event) => { if (!imageMention || mentionOptions.length === 0) return; if (event.key === "ArrowDown") { event.preventDefault(); setActiveMentionIndex((current) => (current + 1) % mentionOptions.length); } else if (event.key === "ArrowUp") { event.preventDefault(); setActiveMentionIndex((current) => (current - 1 + mentionOptions.length) % mentionOptions.length); } else if (event.key === "Enter" || event.key === "Tab") { event.preventDefault(); insertImageMention(mentionOptions[activeMentionIndex].number); } else if (event.key === "Escape") { event.preventDefault(); setImageMention(null); } }} placeholder={mode === "edit" ? "描述编辑要求，可输入 @ 选择参考图" : "描述你想生成或编辑的画面"} rows={4} />
            {mode === "edit" && imageMention && mentionOptions.length > 0 && <div className="absolute inset-x-2 bottom-2 z-20 max-h-52 overflow-auto rounded-2xl border border-sky-200 bg-white p-1.5 shadow-xl dark:border-slate-600 dark:bg-slate-800" role="listbox" aria-label="选择参考图片"><div className="px-2 py-1 text-[11px] font-bold text-slate-400">选择图片引用</div>{mentionOptions.map(({ image, number }, index) => <button key={image.id} className={`flex w-full items-center gap-2 rounded-xl px-2 py-1.5 text-left text-xs font-bold transition ${index === activeMentionIndex ? "bg-sky-100 text-sky-800 dark:bg-sky-950/70 dark:text-sky-200" : "text-slate-700 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-700"}`} type="button" role="option" aria-selected={index === activeMentionIndex} onMouseDown={(event) => event.preventDefault()} onClick={() => insertImageMention(number)}><img className="h-9 w-9 rounded-lg object-cover" src={image.preview} alt="" /><span>@{number} {number === 1 ? "主图" : "参考图"}</span><span className="ml-auto max-w-32 truncate font-normal text-slate-400">{image.file.name}</span></button>)}</div>}
          </div>
          {mode === "edit" && sourceImages.length > 0 && <p className="text-xs font-semibold text-slate-500 dark:text-slate-400">输入 @ 可指定图片，例如 @2；提交时所有已上传图片都会作为输入。</p>}
        </label>
        <div className="grid gap-3 sm:gap-4">
          <div className="form-control grid min-w-0 gap-1.5 sm:gap-2"><div className="flex items-center justify-between gap-3"><span className="label-text font-bold text-slate-800">模型</span><button className="btn btn-ghost btn-sm shrink-0 rounded-xl px-3 font-bold text-sky-700 dark:text-sky-300" type="button" aria-expanded={isManagingModels} onClick={() => setIsManagingModels((current) => !current)} disabled={isModelsLoading || isModelsSaving}>{isManagingModels ? "完成" : "管理"}</button></div><select className="select select-bordered h-11 min-w-0 w-full truncate rounded-2xl border-slate-200 bg-white/80 px-3 text-sm shadow-inner shadow-slate-100 transition focus:border-sky-400 focus:outline-sky-200 dark:border-slate-600 dark:bg-slate-950/60 dark:shadow-[inset_0_2px_6px_rgba(0,0,0,0.3)] dark:focus:border-sky-400 dark:focus:outline-sky-900 sm:h-12 sm:px-4 sm:text-base" value={model} onChange={(event) => onModelChange(event.target.value)} disabled={isModelsLoading}>{modelOptions.map((opt) => <option key={opt.id} value={opt.value}>{opt.label}</option>)}</select>{modelsError && !isManagingModels && <div className="rounded-xl border border-amber-200 bg-amber-50 px-3 py-2 text-xs font-bold text-amber-700 dark:border-amber-900/70 dark:bg-amber-950/30 dark:text-amber-300" role="status">模型列表暂时无法加载，当前仍可使用固定模型。</div>}{isManagingModels && <ModelManagerPanel models={modelOptions} isSaving={isModelsSaving} error={modelsError} onAdd={onAddModel} onDelete={onDeleteModel} onAdded={onModelChange} />}</div>
          <div className="grid grid-cols-2 gap-2.5 sm:gap-3"><label className="form-control grid min-w-0 gap-1.5 sm:gap-2"><span className="label-text font-bold text-slate-800">尺寸</span><select className="select select-bordered h-11 min-w-0 w-full truncate rounded-2xl border-slate-200 bg-white/80 px-2 text-sm shadow-inner shadow-slate-100 transition focus:border-sky-400 focus:outline-sky-200 dark:border-slate-600 dark:bg-slate-950/60 dark:shadow-[inset_0_2px_6px_rgba(0,0,0,0.3)] dark:focus:border-sky-400 dark:focus:outline-sky-900 sm:h-12 sm:px-3 sm:text-base" value={currentAspectRatio} onChange={(event) => mode === "generate" ? onGenerateAspectRatioChange(event.target.value) : onEditAspectRatioChange(event.target.value)}>{currentSizeOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</select></label><label className="form-control grid min-w-0 gap-1.5 sm:gap-2"><span className="label-text font-bold text-slate-800">分辨率</span><select className="select select-bordered h-11 min-w-0 w-full truncate rounded-2xl border-slate-200 bg-white/80 px-2 text-sm shadow-inner shadow-slate-100 transition focus:border-sky-400 focus:outline-sky-200 dark:border-slate-600 dark:bg-slate-950/60 dark:shadow-[inset_0_2px_6px_rgba(0,0,0,0.3)] dark:focus:border-sky-400 dark:focus:outline-sky-900 sm:h-12 sm:px-3 sm:text-base" value={currentResolution} onChange={(event) => mode === "generate" ? onGenerateResolutionChange(event.target.value) : onEditResolutionChange(event.target.value)}>{currentResolutionOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</select></label></div>
        </div>
        {mode === "edit" && <div className="grid gap-2.5"><div className="flex items-center justify-between gap-3"><span className="label-text font-bold text-slate-800 dark:text-slate-100">编辑图片</span><span className="text-xs font-bold text-slate-400">{sourceImages.length}/{maxSourceImages}</span></div><label className="relative flex min-h-28 max-h-[min(75svh,42rem)] cursor-pointer items-center justify-center overflow-hidden rounded-3xl border border-dashed border-sky-300/70 bg-sky-50/60 text-sm font-bold text-sky-700/80 transition hover:bg-sky-50 dark:border-sky-700/70 dark:bg-slate-950/50 dark:text-sky-300 dark:hover:bg-slate-900"><input accept="image/*" className="hidden" type="file" onChange={(event) => { onMainImageChange(event.currentTarget.files?.[0] ?? null); event.currentTarget.value = ""; }} />{sourceImages[0] ? <><img className="max-h-[min(75svh,42rem)] w-full object-contain" src={sourceImages[0].preview} alt="待编辑主图预览，点击图片可更换" /><span className="absolute left-3 top-3 rounded-lg bg-slate-950/70 px-2 py-1 text-xs font-black text-white">@1 主图</span></> : <span className="flex min-h-28 items-center justify-center gap-2 px-4 py-3"><span className="text-lg">＋</span><span>上传主图</span></span>}</label>{sourceImages.length > 1 && <div className="grid grid-cols-3 gap-2">{sourceImages.slice(1).map((image, index) => { const number = index + 2; return <div className="relative overflow-hidden rounded-2xl border border-slate-200 bg-slate-100 dark:border-slate-600 dark:bg-slate-900" key={image.id}><img className="aspect-square w-full object-cover" src={image.preview} alt={`参考图 ${number}`} /><span className="absolute left-1.5 top-1.5 rounded-md bg-slate-950/70 px-1.5 py-0.5 text-[10px] font-black text-white">@{number}</span><button className="absolute right-1.5 top-1.5 h-6 w-6 rounded-full bg-slate-950/70 text-sm font-black leading-none text-white hover:bg-red-600" type="button" aria-label={`删除参考图 ${number}`} onClick={() => onRemoveSourceImage(image.id)}>×</button></div>; })}</div>}{sourceImages.length < maxSourceImages && <label className="flex min-h-12 cursor-pointer items-center justify-center gap-2 rounded-2xl border border-dashed border-slate-300 bg-white/60 text-sm font-bold text-slate-500 transition hover:border-sky-300 hover:bg-sky-50 hover:text-sky-700 dark:border-slate-600 dark:bg-slate-950/30 dark:text-slate-400 dark:hover:border-sky-700 dark:hover:bg-slate-900 dark:hover:text-sky-300"><input accept="image/*" className="hidden" type="file" multiple onChange={(event) => { onReferenceImagesChange(Array.from(event.currentTarget.files ?? [])); event.currentTarget.value = ""; }} /><span className="text-lg">＋</span><span>添加参考图（最多 {maxSourceImages - 1} 张）</span></label>}{submitButton}</div>}
        {mode === "generate" && submitButton}
      </form>
    </section>
  );
}
