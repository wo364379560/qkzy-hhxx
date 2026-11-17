/// <reference types="vite/client" />

declare module '*.mp3' {
  const src: string
  export default src
}

declare module '*.m4a' {
  const src: string
  export default src
}

interface ImportMetaEnv {
  readonly VITE_IM_SDK_APP_ID?: string
  readonly VITE_CHAT_API_BASE_URL?: string
  readonly VITE_API_BASE_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

interface Window {
  __ELECTRON__?: boolean
  desktop?: {
    readAccountVault: () => Promise<string>
    writeAccountVault: (raw: string) => Promise<boolean>
    saveFile: (payload: { name: string; buffer: ArrayBuffer }) => Promise<string | null>
    saveFileAuto: (payload: { name: string; buffer: ArrayBuffer }) => Promise<string | null>
    openPath: (filePath: string) => Promise<void>
    showInFolder: (filePath: string) => Promise<void>
    putToCos: (payload: { url: string; filePath: string; contentType?: string }) => Promise<boolean>
    captureScreen: () => Promise<Uint8Array | ArrayBuffer>
  }
}

declare module 'tim-upload-plugin' {
  const plugin: unknown
  export default plugin
}
