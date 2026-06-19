# Agentic Workflow — BNI Finance Hub

## Entry Points

| Scenario | Command |
| --- | --- |
| Raw task atau fitur baru | `/agentic-start "<task>"` |
| Kerjakan task epic yang ada | `/task-work EPIC-XXX <n>` |
| Loop semua task on-progress | `/epic-loop` |
| Generate / update dokumentasi | `/workflows:claude-codex-docs <path>` |

## Status Vocabulary

| Status | Makna |
| --- | --- |
| `backlog` | Direncanakan, belum dimulai |
| `on-progress` | Siap untuk otomasi |
| `coding` | Implementasi berjalan |
| `review` | Review gate berjalan |
| `testing` | Test gate berjalan |
| `deploying-dev` | Deploy DEV berjalan |
| `ready-for-qa` | Otomasi selesai, butuh QA manusia |
| `blocked` | Butuh keputusan manusia |
| `done` | Selesai |

## Epics

- [`docs/epics/EPIC-000-project-bootstrap.md`](epics/EPIC-000-project-bootstrap.md) — bootstrap agentic workflow
- [`docs/epics/EPIC-001-xendit-self-payment.md`](epics/EPIC-001-xendit-self-payment.md) — Self Payment Mode via Xendit

## Safety Boundaries

- Tidak ada deploy produksi dari otomasi.
- Tidak ada force-push dari otomasi.
- Selalu review `git diff` sebelum push atau PR.
- Tidak ada secrets dalam commit.
- Service Role key & Xendit Secret Key **tidak boleh** ada di variabel `VITE_*`.
