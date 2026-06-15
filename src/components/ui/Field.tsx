import { forwardRef, type InputHTMLAttributes, type ReactNode, type SelectHTMLAttributes, type TextareaHTMLAttributes } from 'react'
import { cn } from '@/lib/cn'

const base =
  'w-full rounded-xl border border-ink-200 bg-white px-3.5 text-sm text-ink-900 placeholder:text-ink-400 transition-colors focus-ring hover:border-ink-300 focus:border-brand-400 disabled:bg-ink-50 disabled:text-ink-400'

interface LabelWrapProps {
  label?: ReactNode
  hint?: ReactNode
  error?: ReactNode
  required?: boolean
  children: ReactNode
  className?: string
}

export function Field({ label, hint, error, required, children, className }: LabelWrapProps) {
  return (
    <label className={cn('block', className)}>
      {label && (
        <span className="mb-1.5 flex items-center gap-1 text-[13px] font-medium text-ink-700">
          {label}
          {required && <span className="text-brand-500">*</span>}
        </span>
      )}
      {children}
      {error ? (
        <span className="mt-1 block text-xs text-red-600">{error}</span>
      ) : hint ? (
        <span className="mt-1 block text-xs text-ink-400">{hint}</span>
      ) : null}
    </label>
  )
}

export const Input = forwardRef<HTMLInputElement, InputHTMLAttributes<HTMLInputElement>>(
  ({ className, ...props }, ref) => (
    <input ref={ref} className={cn(base, 'h-10', className)} {...props} />
  ),
)
Input.displayName = 'Input'

export const Select = forwardRef<HTMLSelectElement, SelectHTMLAttributes<HTMLSelectElement>>(
  ({ className, children, ...props }, ref) => (
    <div className="relative">
      <select
        ref={ref}
        className={cn(base, 'h-10 appearance-none pr-9 cursor-pointer', className)}
        {...props}
      >
        {children}
      </select>
      <svg
        className="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-400"
        viewBox="0 0 20 20"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.6"
      >
        <path d="M6 8l4 4 4-4" strokeLinecap="round" strokeLinejoin="round" />
      </svg>
    </div>
  ),
)
Select.displayName = 'Select'

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaHTMLAttributes<HTMLTextAreaElement>>(
  ({ className, ...props }, ref) => (
    <textarea ref={ref} className={cn(base, 'py-2.5 min-h-[88px] resize-y', className)} {...props} />
  ),
)
Textarea.displayName = 'Textarea'
