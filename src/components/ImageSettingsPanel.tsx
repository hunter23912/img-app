import type { ImageSettings } from "../types/image";
import { useState } from "react";

interface ImageSettingsPanelProps {
  settings: ImageSettings;
  isLoading: boolean;
  isSaving: boolean;
  isDirty: boolean;
  error: string;
  onChange: (next: Partial<ImageSettings>) => void;
  onSave: () => void | Promise<void>;
  onReset: () => void | Promise<void>;
}

export function ImageSettingsPanel({
  settings,
  isLoading,
  isSaving,
  isDirty,
  error,
  onChange,
  onSave,
  onReset,
}: ImageSettingsPanelProps) {
  const disabled = isLoading || isSaving;

  return (
    <details className="card rounded-[1.25rem] border border-white/70 bg-white/75 shadow-[0_18px_60px_rgba(15,23,42,0.08)] backdrop-blur dark:border-slate-600/50 dark:bg-slate-800/80 dark:shadow-[0_18px_60px_rgba(0,0,0,0.32)] sm:rounded-[1.4rem]">
      <summary className="flex cursor-pointer list-none items-center justify-between gap-3 p-4 font-black text-slate-900 outline-none focus-visible:ring-2 focus-visible:ring-sky-400 focus-visible:ring-offset-2 dark:text-slate-100 dark:focus-visible:ring-offset-slate-950 [&::-webkit-details-marker]:hidden sm:p-5">
        <span className="flex min-w-0 items-center gap-2">
          <span>图片服务配置</span>
          {isDirty && !disabled && (
            <span
              className="h-2 w-2 rounded-full bg-amber-500"
              aria-label="有未保存修改"
            />
          )}
        </span>
        <span
          className="text-xs font-bold text-slate-400 transition-transform [details[open]_&]:rotate-180"
          aria-hidden="true"
        >
          ⌄
        </span>
      </summary>

      <ImageSettingsFields
        settings={settings}
        disabled={disabled}
        isSaving={isSaving}
        isDirty={isDirty}
        error={error}
        onChange={onChange}
        onSave={onSave}
        onReset={onReset}
      />
    </details>
  );
}

interface ImageSettingsFieldsProps {
  settings: ImageSettings;
  disabled: boolean;
  isSaving: boolean;
  isDirty: boolean;
  error: string;
  onChange: (next: Partial<ImageSettings>) => void;
  onSave: () => void | Promise<void>;
  onReset: () => void | Promise<void>;
}

export function ImageSettingsFields({
  settings,
  disabled,
  isSaving,
  isDirty,
  error,
  onChange,
  onSave,
  onReset,
}: ImageSettingsFieldsProps) {
  const [clipboardMessage, setClipboardMessage] = useState("");
  const [clipboardError, setClipboardError] = useState("");
  const [isApiKeyEditing, setIsApiKeyEditing] = useState(false);

  function resetClipboardFeedback() {
    setClipboardMessage("");
    setClipboardError("");
  }

  async function handleCopy(value: string, label: string) {
    resetClipboardFeedback();
    if (!value) return;

    try {
      if (!navigator.clipboard?.writeText)
        throw new Error("clipboard unavailable");
      await navigator.clipboard.writeText(value);
      setClipboardMessage(`${label}已复制。`);
    } catch {
      setClipboardError(`无法复制${label}，请检查浏览器剪贴板权限。`);
    }
  }

  async function handlePaste(field: "endpoint" | "api_key", label: string) {
    resetClipboardFeedback();

    try {
      if (!navigator.clipboard?.readText)
        throw new Error("clipboard unavailable");
      const value = (await navigator.clipboard.readText()).trim();
      if (!value) {
        setClipboardError("剪贴板为空。");
        return;
      }
      onChange(field === "endpoint" ? { endpoint: value } : { api_key: value });
      setClipboardMessage(`${label}已粘贴。`);
    } catch {
      setClipboardError(`无法读取剪贴板，请检查浏览器剪贴板权限。`);
    }
  }

  return (
    <div className="grid gap-3 border-t border-slate-200/70 px-4 pb-4 pt-3 dark:border-slate-700/70 sm:gap-4 sm:px-5 sm:pb-5">
      <div className="grid gap-1.5">
        <span className="flex items-center justify-between gap-2">
          <label
            className="text-sm font-bold text-slate-800 dark:text-slate-100"
            htmlFor="image-settings-endpoint"
          >
            Endpoint
          </label>
          <ClipboardActions
            canCopy={Boolean(settings.endpoint)}
            disabled={disabled}
            label="Endpoint"
            onCopy={() => void handleCopy(settings.endpoint, "Endpoint")}
            onPaste={() => void handlePaste("endpoint", "Endpoint")}
          />
        </span>
        <input
          className="input input-bordered h-9 w-full rounded-xl border-slate-200 bg-white/80 px-2.5 text-xs shadow-inner shadow-slate-100 transition focus:border-sky-400 focus:outline-sky-200 dark:border-slate-600 dark:bg-slate-950/60 dark:shadow-[inset_0_2px_6px_rgba(0,0,0,0.3)] dark:focus:border-sky-400 dark:focus:outline-sky-900 sm:h-10 sm:px-3 sm:text-sm"
          id="image-settings-endpoint"
          type="url"
          value={settings.endpoint}
          onChange={(event) => onChange({ endpoint: event.target.value })}
          placeholder="https://example.com"
          disabled={disabled}
          autoComplete="url"
        />
      </div>

      <div className="grid gap-1.5">
        <span className="flex items-center justify-between gap-2">
          <span className="flex min-w-0 items-center gap-2 text-sm font-bold text-slate-800 dark:text-slate-100">
            <label
              className="text-sm font-bold"
              htmlFor="image-settings-api-key"
            >
              API Key
            </label>
          </span>
          <ClipboardActions
            canCopy={Boolean(settings.api_key)}
            disabled={disabled}
            label="API Key"
            onCopy={() => void handleCopy(settings.api_key, "API Key")}
            onPaste={() => void handlePaste("api_key", "API Key")}
          />
        </span>
        <div className="relative">
          <input
            className="input input-bordered h-9 w-full rounded-xl border-slate-200 bg-white/80 px-2.5 text-xs shadow-inner shadow-slate-100 transition focus:border-sky-400 focus:outline-sky-200 dark:border-slate-600 dark:bg-slate-950/60 dark:shadow-[inset_0_2px_6px_rgba(0,0,0,0.3)] dark:focus:border-sky-400 dark:focus:outline-sky-900 sm:h-10 sm:px-3 sm:text-sm"
            id="image-settings-api-key"
            type="password"
            value={settings.api_key}
            onChange={(event) => onChange({ api_key: event.target.value })}
            onFocus={() => setIsApiKeyEditing(true)}
            onBlur={() => setIsApiKeyEditing(false)}
            placeholder="输入中转站 API Key"
            disabled={disabled}
            autoComplete="off"
            style={settings.api_key && !isApiKeyEditing ? { color: "transparent", caretColor: "transparent" } : undefined}
          />
          {settings.api_key && !isApiKeyEditing && (
            <span
              className="pointer-events-none absolute inset-y-0 left-2.5 flex items-center truncate text-xs font-semibold text-slate-500 sm:left-3 sm:text-sm dark:text-slate-400"
              aria-hidden="true"
              title="API Key 部分摘要"
            >
              {formatApiKeyPreview(settings.api_key)}
            </span>
          )}
        </div>
      </div>

      {(error || clipboardError) && (
        <p
          className="text-sm font-bold text-rose-600 dark:text-rose-300"
          role="alert"
        >
          {error || clipboardError}
        </p>
      )}
      {clipboardMessage && !clipboardError && (
        <p
          className="text-xs font-bold text-emerald-600 dark:text-emerald-300"
          role="status"
        >
          {clipboardMessage}
        </p>
      )}

      <div className="grid grid-cols-2 gap-2">
        <button
          className="btn min-h-7 rounded-xl border border-slate-200 bg-white/80 px-2 text-xs font-black text-slate-700 shadow-sm transition hover:bg-slate-100 disabled:bg-slate-200 dark:border-slate-600 dark:bg-slate-900/60 dark:text-slate-100 dark:hover:bg-slate-700 dark:disabled:bg-slate-800"
          type="button"
          onClick={() => void onReset()}
          disabled={disabled}
        >
          恢复服务端默认值
        </button>
        <button
          className="btn min-h-7 rounded-xl border-0 bg-sky-500 px-2 text-xs font-black text-white shadow-[0_7px_16px_rgba(14,165,233,0.2)] transition hover:bg-sky-600 disabled:bg-slate-300"
          type="button"
          onClick={() => void onSave()}
          disabled={disabled || !isDirty}
        >
          {isSaving ? "保存中..." : "保存配置"}
        </button>
      </div>
    </div>
  );
}

interface ClipboardActionsProps {
  canCopy: boolean;
  disabled: boolean;
  label: string;
  onCopy: () => void;
  onPaste: () => void;
}

function ClipboardActions({
  canCopy,
  disabled,
  label,
  onCopy,
  onPaste,
}: ClipboardActionsProps) {
  return (
    <span className="inline-flex shrink-0 items-center gap-1">
      <button
        className="rounded-md px-1 py-0 text-[7px] font-bold text-sky-600 transition hover:bg-sky-50 disabled:cursor-not-allowed disabled:opacity-40 dark:text-sky-300 dark:hover:bg-sky-950/60"
        type="button"
        onClick={onPaste}
        disabled={disabled}
        aria-label={`粘贴${label}`}
      >
        粘贴
      </button>
      <button
        className="rounded-md px-1 py-0 text-[7px] font-bold text-sky-600 transition hover:bg-sky-50 disabled:cursor-not-allowed disabled:opacity-40 dark:text-sky-300 dark:hover:bg-sky-950/60"
        type="button"
        onClick={onCopy}
        disabled={disabled || !canCopy}
        aria-label={`复制${label}`}
      >
        复制
      </button>
    </span>
  );
}

function formatApiKeyPreview(value: string) {
  if (value.length <= 8) {
    return `${value.slice(0, 2)}••••${value.slice(-2)}`;
  }

  return `${value.slice(0, 6)}••••${value.slice(-4)}`;
}
