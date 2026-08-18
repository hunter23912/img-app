import { useState } from 'react'
import type { FormEvent } from 'react'

import type { ImageProfile, ImageProfileDraft } from '../types/image'

interface ImageProfilesPageProps {
  profiles: ImageProfile[]
  isLoading: boolean
  isSaving: boolean
  error: string
  onCreate: (draft: ImageProfileDraft) => Promise<unknown>
  onUpdate: (id: string, draft: ImageProfileDraft) => Promise<unknown>
  onActivate: (id: string) => Promise<unknown>
  onDelete: (id: string) => Promise<unknown>
}

const emptyDraft: ImageProfileDraft = { name: '', endpoint: '', api_key: '' }

export function ImageProfilesPage({
  profiles,
  isLoading,
  isSaving,
  error,
  onCreate,
  onUpdate,
  onActivate,
  onDelete,
}: ImageProfilesPageProps) {
  const [draft, setDraft] = useState<ImageProfileDraft>(emptyDraft)
  const [editingID, setEditingID] = useState('')
  const [message, setMessage] = useState('')
  const [clipboardMessage, setClipboardMessage] = useState('')
  const disabled = isLoading || isSaving

  function editProfile(profile: ImageProfile) {
    setEditingID(profile.id)
    setDraft({ name: profile.name, endpoint: profile.endpoint, api_key: profile.api_key })
    setMessage('')
    setClipboardMessage('')
  }

  function resetForm() {
    setEditingID('')
    setDraft(emptyDraft)
    setClipboardMessage('')
  }

  async function copyToClipboard(value: string, label: string) {
    setClipboardMessage('')
    if (!value) {
      setClipboardMessage(`${label}为空。`)
      return
    }
    try {
      if (!navigator.clipboard?.writeText) throw new Error('clipboard unavailable')
      await navigator.clipboard.writeText(value)
      setClipboardMessage(`${label}已复制。`)
    } catch {
      setClipboardMessage(`无法复制${label}，请检查浏览器剪贴板权限。`)
    }
  }

  async function pasteFromClipboard(field: 'endpoint' | 'api_key', label: string) {
    setClipboardMessage('')
    try {
      if (!navigator.clipboard?.readText) throw new Error('clipboard unavailable')
      const value = (await navigator.clipboard.readText()).trim()
      if (!value) {
        setClipboardMessage('剪贴板为空。')
        return
      }
      setDraft((current) => ({ ...current, [field]: field === 'endpoint' ? normalizeEndpoint(value) : value }))
      setClipboardMessage(`${label}已粘贴。`)
    } catch {
      setClipboardMessage('无法读取剪贴板，请检查浏览器剪贴板权限。')
    }
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setMessage('')
    const normalizedDraft = { ...draft, endpoint: normalizeEndpoint(draft.endpoint) }
    setDraft(normalizedDraft)
    try {
      if (editingID) {
        await onUpdate(editingID, normalizedDraft)
        setMessage('中转站配置已保存。')
      } else {
        await onCreate(normalizedDraft)
        setMessage('中转站配置已添加。')
      }
      resetForm()
    } catch {
      // The hook exposes the server error below the form.
    }
  }

  async function handleActivate(id: string) {
    setMessage('')
    try {
      await onActivate(id)
      setMessage('当前中转站已切换。')
    } catch {
      // The hook exposes the server error below the form.
    }
  }

  async function handleDelete(profile: ImageProfile) {
    if (!window.confirm(`确定删除“${profile.name}”吗？`)) return
    setMessage('')
    try {
      await onDelete(profile.id)
      if (editingID === profile.id) resetForm()
      setMessage('中转站配置已删除。')
    } catch {
      // The hook exposes the server error below the form.
    }
  }

  function handleEndpointBlur() {
    setDraft((current) => ({ ...current, endpoint: normalizeEndpoint(current.endpoint) }))
  }

  return (
    <section className="grid gap-3 sm:gap-4">
      <div className="card rounded-[1.75rem] border border-white/70 bg-white/60 p-2 shadow-[0_18px_60px_rgba(15,23,42,0.08)] backdrop-blur dark:border-slate-600/50 dark:bg-slate-800/60 dark:shadow-[0_18px_60px_rgba(0,0,0,0.32)] sm:rounded-[2rem] sm:p-3">
        <div className="flex items-center justify-between gap-2 px-2 pb-2 pt-1 sm:px-2.5 sm:pt-0.5">
          <h2 className="text-lg font-black text-slate-950 dark:text-slate-100">配置列表</h2>
          {clipboardMessage && <p className="text-xs font-bold text-emerald-600 dark:text-emerald-300" role="status">{clipboardMessage}</p>}
        </div>
        <div className="grid gap-2.5">
          {isLoading ? (
            <div className="rounded-[1.5rem] p-5 text-center text-sm font-bold text-slate-500 sm:rounded-[1.75rem]">读取配置中...</div>
          ) : profiles.length === 0 ? (
            <div className="rounded-[1.5rem] p-5 text-center text-sm font-bold text-slate-500 dark:text-slate-400 sm:rounded-[1.75rem]">
              还没有保存的中转站配置。
            </div>
          ) : (
            profiles.map((profile) => (
              <article
                className={`card rounded-[1.5rem] border p-3 shadow-sm transition sm:rounded-[1.75rem] sm:p-4 ${profile.is_active ? 'border-sky-300 bg-sky-50/80 dark:border-sky-700 dark:bg-sky-950/30' : 'border-white/70 bg-white/75 dark:border-slate-600/50 dark:bg-slate-800/80'}`}
                key={profile.id}
              >
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <h3 className="truncate font-black text-slate-900 dark:text-slate-100">{profile.name}</h3>
                  </div>
                </div>
                <div className="flex shrink-0 gap-1">
                  {profile.is_active ? (
                    <button className="btn h-7 min-h-7 w-16 cursor-default justify-center rounded-xl border-0 bg-slate-200 px-2 text-[8px] font-black text-slate-500 opacity-100 hover:bg-slate-200 disabled:opacity-100 dark:bg-slate-700 dark:text-slate-400 dark:hover:bg-slate-700" type="button" disabled aria-label={`${profile.name}已启用`}>
                      启用中
                    </button>
                  ) : (
                    <button className="btn h-7 min-h-7 w-16 justify-center rounded-xl border-0 bg-sky-500 px-2 text-[8px] font-black text-white hover:bg-sky-600 disabled:bg-slate-300" type="button" onClick={() => void handleActivate(profile.id)} disabled={disabled} aria-label={`启用${profile.name}`}>
                      启用
                    </button>
                  )}
                  <button className="btn h-7 min-h-7 w-16 justify-center rounded-xl border border-slate-200 bg-white/80 px-2 text-[8px] font-black text-slate-700 hover:bg-slate-100 dark:border-slate-600 dark:bg-slate-900/60 dark:text-slate-100" type="button" onClick={() => editProfile(profile)} disabled={disabled}>
                    编辑
                  </button>
                  <button className="btn h-7 min-h-7 w-16 justify-center rounded-xl border border-rose-200 bg-white/80 px-2 text-[8px] font-black text-rose-600 hover:bg-rose-50 disabled:opacity-50 dark:border-rose-900 dark:bg-slate-900/60 dark:text-rose-300" type="button" onClick={() => void handleDelete(profile)} disabled={disabled || profile.is_active}>
                    删除
                  </button>
                </div>
              </div>
              <div className="mt-2 grid gap-1.5 sm:grid-cols-2">
                <button className="flex min-w-0 items-center gap-2 rounded-2xl border border-slate-200/70 bg-white/80 px-2.5 py-1.5 text-left transition hover:border-sky-300 hover:bg-sky-50/70 active:scale-[0.99] disabled:cursor-not-allowed disabled:opacity-60 dark:border-slate-700/70 dark:bg-slate-900/50 dark:hover:border-sky-700 dark:hover:bg-sky-950/30" type="button" onClick={() => void copyToClipboard(profile.endpoint, `${profile.name}端点`)} disabled={disabled} title="点击复制端点" aria-label={`点击复制${profile.name}的端点`}>
                  <span className="shrink-0 rounded-lg bg-sky-100 px-1.5 py-0.5 text-[10px] font-black text-sky-700 dark:bg-sky-950/70 dark:text-sky-300">端点</span>
                  <span className="min-w-0 break-all text-left text-xs font-semibold leading-snug text-slate-600 dark:text-slate-300">{profile.endpoint}</span>
                </button>
                <button className="flex min-w-0 items-center gap-2 rounded-2xl border border-slate-200/70 bg-white/80 px-2.5 py-1.5 text-left transition hover:border-sky-300 hover:bg-sky-50/70 active:scale-[0.99] disabled:cursor-not-allowed disabled:opacity-60 dark:border-slate-700/70 dark:bg-slate-900/50 dark:hover:border-sky-700 dark:hover:bg-sky-950/30" type="button" onClick={() => void copyToClipboard(profile.api_key, `${profile.name}密钥`)} disabled={disabled || !profile.api_key} title="点击复制密钥" aria-label={`点击复制${profile.name}的密钥`}>
                  <span className="shrink-0 rounded-lg bg-slate-200 px-1.5 py-0.5 text-[10px] font-black text-slate-600 dark:bg-slate-700 dark:text-slate-300">密钥</span>
                  <span className="min-w-0 truncate text-left text-xs font-semibold text-slate-600 dark:text-slate-300" title={profile.api_key || 'API Key 未填写'}>{formatApiKeyPreview(profile.api_key)}</span>
                </button>
              </div>
              </article>
            ))
          )}
        </div>
      </div>

      <form className="card grid gap-3 rounded-[1.5rem] border border-white/70 bg-white/75 p-4 shadow-[0_18px_60px_rgba(15,23,42,0.08)] backdrop-blur dark:border-slate-600/50 dark:bg-slate-800/80 dark:shadow-[0_18px_60px_rgba(0,0,0,0.32)] sm:gap-4 sm:rounded-[1.75rem] sm:p-5" onSubmit={(event) => void handleSubmit(event)}>
        <div className="mx-auto grid w-full max-w-2xl gap-3 sm:gap-4">
          <div className="flex items-center justify-between gap-2">
            <h2 className="font-black text-slate-900 dark:text-slate-100">{editingID ? '编辑配置' : '新增配置'}</h2>
            {editingID && <button className="text-xs font-bold text-slate-500 hover:text-sky-600" type="button" onClick={resetForm} disabled={disabled}>取消编辑</button>}
          </div>
          <label className="grid gap-1.5 text-sm font-bold text-slate-800 dark:text-slate-100" htmlFor="image-profile-name">
            名称
            <input id="image-profile-name" className="input input-bordered h-10 w-full rounded-2xl border-slate-200 bg-white/80 px-3 text-sm dark:border-slate-600 dark:bg-slate-950/60" value={draft.name} onChange={(event) => setDraft({ ...draft, name: event.target.value })} placeholder="留空自动命名" disabled={disabled} />
          </label>
          <div className="grid gap-1.5">
            <div className="flex items-center justify-between gap-2">
              <label className="text-sm font-bold text-slate-800 dark:text-slate-100" htmlFor="image-profile-endpoint">Endpoint</label>
              <ClipboardActions canCopy={Boolean(draft.endpoint)} disabled={disabled} label="Endpoint" onCopy={() => void copyToClipboard(draft.endpoint, 'Endpoint')} onPaste={() => void pasteFromClipboard('endpoint', 'Endpoint')} />
            </div>
            <input id="image-profile-endpoint" className="input input-bordered h-10 w-full rounded-2xl border-slate-200 bg-white/80 px-3 text-sm dark:border-slate-600 dark:bg-slate-950/60" type="text" inputMode="url" value={draft.endpoint} onChange={(event) => setDraft({ ...draft, endpoint: event.target.value })} onBlur={handleEndpointBlur} placeholder="example.com 或 https://example.com" disabled={disabled} required />
          </div>
          <div className="grid gap-1.5">
            <div className="flex items-center justify-between gap-2">
              <label className="text-sm font-bold text-slate-800 dark:text-slate-100" htmlFor="image-profile-api-key">API Key</label>
              <ClipboardActions canCopy={Boolean(draft.api_key)} disabled={disabled} label="API Key" onCopy={() => void copyToClipboard(draft.api_key, 'API Key')} onPaste={() => void pasteFromClipboard('api_key', 'API Key')} />
            </div>
            <input id="image-profile-api-key" className="input input-bordered h-10 w-full rounded-2xl border-slate-200 bg-white/80 px-3 text-sm dark:border-slate-600 dark:bg-slate-950/60" type="text" value={draft.api_key} onChange={(event) => setDraft({ ...draft, api_key: event.target.value })} placeholder="输入中转站 API Key" disabled={disabled} autoComplete="off" />
          </div>
          {(error || message) && <p className={`text-sm font-bold ${error ? 'text-rose-600 dark:text-rose-300' : 'text-emerald-600 dark:text-emerald-300'}`} role={error ? 'alert' : 'status'}>{error || message}</p>}
          <button className="btn min-h-10 w-full rounded-2xl border-0 bg-sky-500 font-black text-white hover:bg-sky-600 disabled:bg-slate-300 sm:mx-auto sm:max-w-xs" type="submit" disabled={disabled}>
            {isSaving ? '保存中...' : editingID ? '保存配置' : '添加配置'}
          </button>
        </div>
      </form>
    </section>
  )
}

interface ClipboardActionsProps {
  canCopy: boolean
  disabled: boolean
  label: string
  onCopy: () => void
  onPaste: () => void
}

function ClipboardActions({ canCopy, disabled, label, onCopy, onPaste }: ClipboardActionsProps) {
  return (
    <span className="inline-flex items-center gap-0.5">
      <button className="min-h-7 min-w-8 rounded-xl px-1 py-0 text-[10px] font-bold text-sky-600 transition hover:bg-sky-50 disabled:cursor-not-allowed disabled:opacity-40 dark:text-sky-300 dark:hover:bg-sky-950/60" type="button" onClick={onPaste} disabled={disabled} aria-label={`粘贴${label}`}>
        粘贴
      </button>
      <button className="min-h-7 min-w-8 rounded-xl px-1 py-0 text-[10px] font-bold text-sky-600 transition hover:bg-sky-50 disabled:cursor-not-allowed disabled:opacity-40 dark:text-sky-300 dark:hover:bg-sky-950/60" type="button" onClick={onCopy} disabled={disabled || !canCopy} aria-label={`复制${label}`}>
        复制
      </button>
    </span>
  )
}

function formatApiKeyPreview(value: string) {
  if (!value) return 'API Key 未填写'
  if (value.length <= 12) return value
  return `${value.slice(0, 6)}••••••••${value.slice(-4)}`
}

function normalizeEndpoint(value: string) {
  const endpoint = value.trim()
  if (!endpoint || endpoint.includes('://')) return endpoint
  return `https://${endpoint}`
}
