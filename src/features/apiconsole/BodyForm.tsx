/**
 * Form editor for a request body.
 *
 * A raw JSON textarea asks the reader to already know the field names AND JSON
 * syntax before they can send anything — a misplaced comma fails the request
 * before it leaves the browser. One labelled input per field removes both
 * requirements. The raw JSON stays one click away for whoever prefers it.
 *
 * Fields come from the OpenAPI request schema (see scripts/build-api-collection.mjs),
 * so this form covers every endpoint without a per-endpoint layout.
 */

import { useMemo } from 'react'
import type { ConsoleBodyField } from '@/services/apiConsole'

/** `customerPhone` → `Customer Phone`, `dryRun` → `Dry Run`. */
export function labelFor(name: string): string {
  return name
    .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
    .replace(/[_-]+/g, ' ')
    .replace(/^\w/, (c) => c.toUpperCase())
}

/** Formats a number for display without losing precision. */
function asText(value: unknown): string {
  if (value === undefined || value === null) return ''
  if (typeof value === 'object') return JSON.stringify(value, null, 2)
  return String(value)
}

interface Props {
  fields: ConsoleBodyField[]
  /** Current body as parsed JSON. Invalid JSON is handled by the caller. */
  value: Record<string, unknown>
  onChange: (next: Record<string, unknown>) => void
}

export function BodyForm({ fields, value, onChange }: Props) {
  // Required first: what the endpoint refuses to work without should be what
  // the eye lands on.
  const ordered = useMemo(
    () => [...fields].sort((a, b) => Number(b.required) - Number(a.required)),
    [fields],
  )

  const set = (name: string, next: unknown) => {
    const out = { ...value }
    // An empty optional field is omitted rather than sent as "": the server
    // treats a missing key as "use the default", but an empty string as a real
    // value — which is how you get an invoice with a blank customer name.
    if (next === undefined || next === '') delete out[name]
    else out[name] = next
    onChange(out)
  }

  return (
    <div className="space-y-3">
      {ordered.map((field) => (
        <div key={field.name} className="grid gap-1 sm:grid-cols-[200px_minmax(0,1fr)] sm:items-start">
          <label
            htmlFor={`body-${field.name}`}
            className="pt-1.5 text-sm text-ink-700 sm:text-right"
          >
            {field.label ?? labelFor(field.name)}
            {field.required && <span className="ml-0.5 text-red-500">*</span>}
            <span className="ml-1.5 font-mono text-[10px] text-ink-400">{field.name}</span>
          </label>
          <div className="min-w-0">
            <FieldInput field={field} value={value[field.name]} onChange={(v) => set(field.name, v)} />
            {field.description && (
              <p className="mt-0.5 text-xs text-ink-400">{field.description}</p>
            )}
          </div>
        </div>
      ))}
    </div>
  )
}

const inputClass =
  'w-full rounded-lg border border-ink-200 px-3 py-1.5 text-sm focus:border-brand-500 focus:outline-none'

function FieldInput({
  field,
  value,
  onChange,
}: {
  field: ConsoleBodyField
  value: unknown
  onChange: (next: unknown) => void
}) {
  const id = `body-${field.name}`

  if (field.type === 'boolean') {
    const on = value === undefined ? Boolean(field.default) : Boolean(value)
    return (
      <button
        id={id}
        type="button"
        role="switch"
        aria-checked={on}
        aria-label={field.label ?? labelFor(field.name)}
        onClick={() => onChange(!on)}
        className={`relative mt-1 inline-flex h-6 w-11 flex-shrink-0 items-center rounded-full transition-colors ${
          on ? 'bg-emerald-500' : 'bg-ink-200'
        }`}
      >
        <span
          className={`inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform ${
            on ? 'translate-x-6' : 'translate-x-1'
          }`}
        />
      </button>
    )
  }

  if (field.enum?.length) {
    return (
      <select
        id={id}
        value={asText(value)}
        onChange={(e) => onChange(e.target.value)}
        className={`${inputClass} bg-white`}
      >
        {!field.required && <option value="">— kosong —</option>}
        {field.enum.map((v) => (
          <option key={v} value={v}>
            {v}
          </option>
        ))}
      </select>
    )
  }

  // Only the Paper.id callback has nested objects; a small JSON box beats
  // inventing a nested form for one endpoint.
  if (field.complex) {
    return (
      <textarea
        id={id}
        rows={4}
        spellCheck={false}
        value={asText(value)}
        onChange={(e) => {
          try {
            onChange(JSON.parse(e.target.value))
          } catch {
            // Keep the keystrokes while the JSON is mid-edit and invalid.
            onChange(e.target.value)
          }
        }}
        className={`${inputClass} bg-ink-900 font-mono text-xs text-ink-100`}
      />
    )
  }

  if (field.type === 'integer' || field.type === 'number') {
    return (
      <input
        id={id}
        type="number"
        value={asText(value)}
        placeholder={field.default !== undefined ? `bawaan: ${String(field.default)}` : ''}
        onChange={(e) => onChange(e.target.value === '' ? undefined : Number(e.target.value))}
        className={inputClass}
      />
    )
  }

  return (
    <input
      id={id}
      type={field.format === 'date' ? 'date' : 'text'}
      value={asText(value)}
      placeholder={field.default !== undefined ? `bawaan: ${String(field.default)}` : ''}
      onChange={(e) => onChange(e.target.value)}
      className={inputClass}
    />
  )
}
