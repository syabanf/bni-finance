/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_USE_MOCK: string
  readonly VITE_SUPABASE_URL: string
  readonly VITE_SUPABASE_ANON_KEY: string
  readonly VITE_BNI_VM_API_URL: string
  readonly VITE_PAPER_ID_API_URL: string
  /** Optional demo account for the "Login Cepat" button on deployed demos. */
  readonly VITE_DEMO_EMAIL: string
  readonly VITE_DEMO_PASSWORD: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
