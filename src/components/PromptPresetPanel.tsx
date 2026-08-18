import { useState } from "react";

import { usePromptPresets } from "../hooks/usePromptPresets";
import type {
  ImageMode,
  MessageTone,
  PromptApplyMode,
  PromptPreset,
  PromptPresetDraft,
  PromptPresetScope,
} from "../types/image";
import {
  maxPresetNameLength,
  maxPresetPromptLength,
  maxPromptPresets,
} from "../utils/promptPresets";

interface PromptPresetPanelProps {
  mode: ImageMode;
  applyMode: PromptApplyMode;
  onApplyModeChange: (mode: PromptApplyMode) => void;
  onApplyPrompt: (prompt: string) => void;
}

interface EditorState {
  kind: "create" | "edit";
  id?: string;
}

interface Feedback {
  tone: MessageTone;
  text: string;
}

const scopeLabels: Record<PromptPresetScope, string> = {
  generate: "文生图",
  edit: "图编辑",
  all: "通用",
};

export function PromptPresetPanel({
  mode,
  applyMode,
  onApplyModeChange,
  onApplyPrompt,
}: PromptPresetPanelProps) {
  const {
    presets,
    storageWarning,
    isLoading,
    createPreset,
    updatePreset,
    deletePreset,
  } = usePromptPresets();
  const [selectedID, setSelectedID] = useState("");
  const [isManaging, setIsManaging] = useState(false);
  const [editor, setEditor] = useState<EditorState | null>(null);
  const [draft, setDraft] = useState<PromptPresetDraft>(emptyDraft(mode));
  const [pendingDeleteID, setPendingDeleteID] = useState("");
  const [feedback, setFeedback] = useState<Feedback | null>(null);

  const visiblePresets = presets.filter(
    (preset) => preset.scope === "all" || preset.scope === mode,
  );
  function selectPreset(preset: PromptPreset) {
    setSelectedID(preset.id);
    setFeedback(null);
    onApplyPrompt(preset.prompt);
  }

  function toggleManager() {
    setIsManaging((current) => !current);
    setEditor(null);
    setPendingDeleteID("");
    setFeedback(null);
  }

  function beginCreate() {
    setDraft(emptyDraft(mode));
    setEditor({ kind: "create" });
    setPendingDeleteID("");
    setFeedback(null);
  }

  function beginEdit(preset: PromptPreset) {
    setDraft({ name: preset.name, prompt: preset.prompt, scope: preset.scope });
    setEditor({ kind: "edit", id: preset.id });
    setPendingDeleteID("");
    setFeedback(null);
  }

  async function savePreset() {
    const result =
      editor?.kind === "edit" && editor.id
        ? await updatePreset(editor.id, draft)
        : await createPreset(draft);

    if (!result.ok) {
      setFeedback({ tone: "error", text: result.error ?? "预设保存失败。" });
      return;
    }

    setEditor(null);
    setFeedback({ tone: "success", text: "预设已保存到服务端。" });
  }

  async function confirmDelete(id: string) {
    const result = await deletePreset(id);
    if (!result.ok) {
      setFeedback({ tone: "error", text: result.error ?? "预设删除失败。" });
      return;
    }

    if (selectedID === id) setSelectedID("");
    if (editor?.id === id) setEditor(null);
    setPendingDeleteID("");
    setFeedback({ tone: "success", text: "预设已删除。" });
  }

  return (
    <div className="grid gap-3" aria-label="提示词预设">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h3 className="text-lg font-bold text-slate-800">提示词预设</h3>
        <div className="flex items-center gap-1">
          <div
            className="inline-flex rounded-xl bg-slate-100 p-1 dark:bg-slate-950/70"
            role="group"
            aria-label="提示词预设应用方式"
          >
            <button
              className={`min-h-7 rounded-lg px-2 text-[11px] font-bold transition focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sky-500 ${
                applyMode === "replace"
                  ? "bg-sky-600 text-white shadow-[0_5px_14px_rgba(2,132,199,0.2)]"
                  : "text-slate-500 hover:text-slate-800 dark:text-slate-400 dark:hover:text-slate-100"
              }`}
              type="button"
              aria-pressed={applyMode === "replace"}
              onClick={() => onApplyModeChange("replace")}
            >
              替换
            </button>
            <button
              className={`min-h-8 rounded-lg px-3 text-xs font-bold transition focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sky-500 ${
                applyMode === "append"
                  ? "bg-sky-600 text-white shadow-[0_5px_14px_rgba(2,132,199,0.2)]"
                  : "text-slate-500 hover:text-slate-800 dark:text-slate-400 dark:hover:text-slate-100"
              }`}
              type="button"
              aria-pressed={applyMode === "append"}
              onClick={() => onApplyModeChange("append")}
            >
              追加
            </button>
          </div>
          <button
            className="btn btn-ghost btn-sm shrink-0 rounded-xl px-3 font-bold text-sky-700 dark:text-sky-300"
            type="button"
            aria-expanded={isManaging}
            onClick={toggleManager}
          >
            {isManaging ? "完成" : "管理"}
          </button>
        </div>
      </div>

      {isLoading ? (
        <p className="text-sm font-medium text-slate-500">正在加载预设...</p>
      ) : visiblePresets.length > 0 ? (
        <div
          className="flex flex-wrap gap-1.5"
          aria-label={`${scopeLabels[mode]}可用预设`}
        >
          {visiblePresets.map((preset) => (
            <button
              key={preset.id}
              className={`min-h-8 max-w-full rounded-full px-2.5 text-xs font-bold transition focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sky-500 ${
                selectedID === preset.id
                  ? "bg-sky-600 text-white shadow-[0_5px_14px_rgba(2,132,199,0.2)]"
                  : "bg-sky-50 text-sky-800 shadow-[0_1px_4px_rgba(14,116,144,0.06)] hover:bg-sky-100 dark:bg-sky-950/70 dark:text-sky-200 dark:shadow-[0_1px_5px_rgba(0,0,0,0.18)] dark:hover:bg-sky-900/80 dark:hover:text-sky-100"
              }`}
              type="button"
              aria-pressed={selectedID === preset.id}
              title={`${preset.name} · ${scopeLabels[preset.scope]}`}
              onClick={() => selectPreset(preset)}
            >
              <span className="block truncate">{preset.name}</span>
            </button>
          ))}
        </div>
      ) : (
        <div className="flex items-center justify-between gap-3 border-y border-slate-200/80 py-3">
          <p className="text-sm font-medium text-slate-500 dark:text-slate-400">
            暂无预设
          </p>
          <button
            className="btn btn-ghost btn-sm shrink-0 rounded-xl font-bold text-sky-700 dark:text-sky-300"
            type="button"
            onClick={() => {
              setIsManaging(true);
              beginCreate();
            }}
          >
            新建
          </button>
        </div>
      )}

      {isManaging && (
        <div className="grid gap-3 border-t border-slate-200/80 pt-3">
          <div className="flex items-center justify-between gap-3">
            <p className="text-xs font-bold text-slate-500">
              {presets.length}/{maxPromptPresets}
            </p>
            <button
              className="btn btn-sm rounded-xl border-0 bg-sky-600 px-4 font-bold text-white hover:bg-sky-700 disabled:bg-slate-300"
              type="button"
              disabled={presets.length >= maxPromptPresets}
              onClick={beginCreate}
            >
              新建
            </button>
          </div>

          {editor && (
            <div className="grid gap-3 bg-slate-50/80 p-3 dark:bg-slate-900/70">
              <label className="grid gap-1.5">
                <span className="text-xs font-bold text-slate-700">名称</span>
                <input
                  className="input input-bordered h-11 w-full rounded-xl border-slate-200 bg-white text-base focus:border-sky-400 focus:outline-sky-200 dark:border-slate-600 dark:bg-slate-950/70 dark:focus:border-sky-400 dark:focus:outline-sky-900"
                  value={draft.name}
                  maxLength={maxPresetNameLength}
                  placeholder="例如：电商产品精修"
                  onChange={(event) =>
                    setDraft((current) => ({
                      ...current,
                      name: event.target.value,
                    }))
                  }
                  onKeyDown={(event) => {
                    if (event.key === "Enter") {
                      event.preventDefault();
                      savePreset();
                    }
                  }}
                />
              </label>

              <label className="grid gap-1.5">
                <span className="text-xs font-bold text-slate-700">
                  适用模式
                </span>
                <select
                  className="select select-bordered h-11 w-full rounded-xl border-slate-200 bg-white text-base focus:border-sky-400 focus:outline-sky-200 dark:border-slate-600 dark:bg-slate-950/70 dark:focus:border-sky-400 dark:focus:outline-sky-900"
                  value={draft.scope}
                  onChange={(event) =>
                    setDraft((current) => ({
                      ...current,
                      scope: event.target.value as PromptPresetScope,
                    }))
                  }
                >
                  <option value="generate">文生图</option>
                  <option value="edit">图编辑</option>
                  <option value="all">通用</option>
                </select>
              </label>

              <label className="grid gap-1.5">
                <span className="flex items-center justify-between gap-3 text-xs font-bold text-slate-700">
                  <span>提示词内容</span>
                  <span className="font-semibold tabular-nums text-slate-500">
                    {draft.prompt.length}/{maxPresetPromptLength}
                  </span>
                </span>
                <textarea
                  className="textarea textarea-bordered min-h-32 w-full resize-y rounded-xl border-slate-200 bg-white text-base leading-relaxed focus:border-sky-400 focus:outline-sky-200 dark:border-slate-600 dark:bg-slate-950/70 dark:focus:border-sky-400 dark:focus:outline-sky-900"
                  value={draft.prompt}
                  maxLength={maxPresetPromptLength}
                  placeholder="填写需要反复使用的完整提示词"
                  rows={5}
                  onChange={(event) =>
                    setDraft((current) => ({
                      ...current,
                      prompt: event.target.value,
                    }))
                  }
                />
              </label>

              <div className="flex flex-wrap gap-2">
                <button
                  className="btn btn-sm rounded-xl border-0 bg-sky-600 px-4 font-bold text-white hover:bg-sky-700"
                  type="button"
                  onClick={savePreset}
                >
                  {editor.kind === "create" ? "保存" : "保存修改"}
                </button>
                <button
                  className="btn btn-ghost btn-sm rounded-xl font-bold text-slate-600"
                  type="button"
                  onClick={() => setEditor(null)}
                >
                  取消
                </button>
              </div>
            </div>
          )}

          {presets.length > 0 ? (
            <div
              className="max-h-72 overflow-y-auto border-y border-slate-200/80"
              aria-label="全部提示词预设"
            >
              {presets.map((preset) => (
                <div
                  key={preset.id}
                  className="border-b border-slate-200/70 py-3 last:border-b-0"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <p className="truncate text-sm font-bold text-slate-900">
                        {preset.name}
                      </p>
                      <p className="mt-1 text-xs font-semibold text-slate-500">
                        {scopeLabels[preset.scope]}
                      </p>
                    </div>
                    <div className="flex shrink-0 gap-1">
                      <button
                        className="btn btn-ghost btn-xs rounded-lg font-bold text-sky-700 dark:text-sky-300"
                        type="button"
                        aria-label={`编辑${preset.name}`}
                        onClick={() => beginEdit(preset)}
                      >
                        编辑
                      </button>
                      <button
                        className="btn btn-ghost btn-xs rounded-lg font-bold text-rose-700"
                        type="button"
                        aria-label={`删除${preset.name}`}
                        onClick={() => {
                          setPendingDeleteID(preset.id);
                          setEditor(null);
                          setFeedback(null);
                        }}
                      >
                        删除
                      </button>
                    </div>
                  </div>

                  {pendingDeleteID === preset.id && (
                    <div className="mt-3 flex flex-wrap items-center gap-2 bg-rose-50 px-3 py-2.5 text-sm text-rose-900">
                      <span className="mr-auto font-bold">
                        确认删除“{preset.name}”？
                      </span>
                      <button
                        className="btn btn-xs rounded-lg border-0 bg-rose-600 font-bold text-white hover:bg-rose-700"
                        type="button"
                        onClick={() => confirmDelete(preset.id)}
                      >
                        确认删除
                      </button>
                      <button
                        className="btn btn-ghost btn-xs rounded-lg font-bold text-rose-800"
                        type="button"
                        onClick={() => setPendingDeleteID("")}
                      >
                        取消
                      </button>
                    </div>
                  )}
                </div>
              ))}
            </div>
          ) : (
            <p className="border-y border-slate-200/80 py-4 text-center text-sm font-medium text-slate-500">
              暂无预设
            </p>
          )}
        </div>
      )}

      {(storageWarning || feedback) && (
        <div
          className={`text-sm font-semibold leading-relaxed [overflow-wrap:anywhere] ${
            storageWarning
              ? "text-amber-800"
              : feedback?.tone === "error"
                ? "text-rose-700"
                : feedback?.tone === "success"
                  ? "text-emerald-700"
                  : "text-sky-700"
          }`}
          role={
            storageWarning || feedback?.tone === "error" ? "alert" : "status"
          }
          aria-live="polite"
        >
          {storageWarning || feedback?.text}
        </div>
      )}
    </div>
  );
}

function emptyDraft(mode: ImageMode): PromptPresetDraft {
  return {
    name: "",
    prompt: "",
    scope: mode,
  };
}
