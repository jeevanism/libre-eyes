/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_UI_ENVIRONMENT_LABEL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
