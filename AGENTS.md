# BNI Finance Hub — Codex Instructions

@~/.agentic-workflows/claude-codex-docs-workflow.md

## Role

Codex handles workflow hygiene in this repo:
- Update status vocabulary in docs
- Normalize stale automation logs
- Review diffs and verify agent handoffs
- Tighten runbooks after Claude Code development

Do NOT run Claude Code slash commands. Translate them into concrete repo steps
or tell the user to run them in Claude Code.

## Stack Notes

- **Frontend**: Vite 5 + React 18 + TypeScript + Tailwind CSS 3 + React Router 6
- **Backend**: Supabase (PostgreSQL + PostgREST + Edge Functions Deno)
- **Payment**: Xendit (Virtual Account BCA/BNI/Mandiri/BRI + QRIS) & Paper.id
- **Hosting**: Vercel (`https://bni-finance-five.vercel.app`)
- **Data source**: BNI Visitor Management API (sync, dev-only via Vite proxy)
- **Auth**: Supabase Auth (mock localStorage in dev)

## Test Gate

```bash
npm run typecheck && npm run build
```

> No unit test suite yet. typecheck + build is the gate.
