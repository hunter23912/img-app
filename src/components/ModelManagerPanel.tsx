import { useState } from 'react'

import type { ModelOption } from '../types/image'

interface ModelManagerPanelProps {
  models: ModelOption[]
  isSaving: boolean
  error: string
  onAdd: (model: string) => Promise<ModelOption>
  onDelete: (id: string) => Promise<void>
  onAdded: (model: string) => void
}

export function ModelManagerPanel({ models, isSaving, error, onAdd, onDelete, onAdded }: ModelManagerPanelProps) {
  const [modelName, setModelName] = useState('')
  const customModels = models.filter((model) => !model.built_in)

  async function handleAdd() {
    const value = modelName.trim()
    if (!value) return
    try {
      const created = await onAdd(value)
      setModelName('')
      onAdded(created.value)
    } catch {
      // The parent hook exposes the server error below the input.
    }
  }

  async function handleDelete(model: ModelOption) {
    if (!window.confirm(`确定删除模型“${model.label}”吗？`)) return
    try {
      await onDelete(model.id)
    } catch {
      // The parent hook exposes the server error below the input.
    }
  }

  return (
    <div className="rounded-2xl border border-sky-200/80 bg-sky-50/70 p-3 dark:border-sky-800/70 dark:bg-sky-950/30">
      <div className="grid gap-2">
        <div className="flex items-center justify-between gap-2">
          <span className="text-sm font-black text-slate-800 dark:text-slate-100">管理模型</span>
          <span className="text-xs font-semibold text-slate-500 dark:text-slate-400">固定模型不可删除</span>
        </div>
        <div className="flex gap-2">
          <input
            className="input input-bordered h-10 min-w-0 flex-1 rounded-xl border-slate-200 bg-white/90 px-3 text-sm dark:border-slate-600 dark:bg-slate-950/60"
            value={modelName}
            onChange={(event) => setModelName(event.target.value)}
            onKeyDown={(event) => { if (event.key === 'Enter') { event.preventDefault(); void handleAdd() } }}
            placeholder="输入模型名，如 vendor/model"
            maxLength={120}
            disabled={isSaving}
          />
          <button className="btn h-10 min-h-10 rounded-xl border-0 bg-sky-500 px-3 text-sm font-black text-white hover:bg-sky-600 disabled:bg-slate-300" type="button" onClick={() => void handleAdd()} disabled={isSaving || !modelName.trim()}>
            添加
          </button>
        </div>
        {customModels.length > 0 && (
          <div className="grid gap-1.5">
            {customModels.map((model) => (
              <div className="flex items-center justify-between gap-2 rounded-xl bg-white/75 px-2.5 py-2 dark:bg-slate-900/60" key={model.id}>
                <span className="min-w-0 truncate text-sm font-semibold text-slate-700 dark:text-slate-200">{model.label}</span>
                <button className="shrink-0 rounded-lg px-2 py-1 text-xs font-bold text-rose-600 hover:bg-rose-50 dark:text-rose-300 dark:hover:bg-rose-950/40" type="button" onClick={() => void handleDelete(model)} disabled={isSaving}>删除</button>
              </div>
            ))}
          </div>
        )}
        {error && <p className="text-xs font-bold text-rose-600 dark:text-rose-300" role="alert">{error}</p>}
      </div>
    </div>
  )
}
