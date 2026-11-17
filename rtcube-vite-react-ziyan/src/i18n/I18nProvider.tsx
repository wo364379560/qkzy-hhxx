import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { MESSAGES, type Lang } from './messages'

type Dict = (typeof MESSAGES)['zh-CN']
type Key = keyof Dict

type I18nValue = {
  lang: Lang
  setLang: (l: Lang) => void
  t: (key: Key) => string
}

const Ctx = createContext<I18nValue | null>(null)
const LANG_KEY = 'cn_ziyan_lang'

export function I18nProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>(() => {
    const saved = localStorage.getItem(LANG_KEY) as Lang | null
    return saved === 'en-US' ? 'en-US' : 'zh-CN'
  })

  const setLang = useCallback((l: Lang) => {
    setLangState(l)
    localStorage.setItem(LANG_KEY, l)
  }, [])

  const t = useCallback((key: Key) => MESSAGES[lang][key] || key, [lang])

  const value = useMemo(() => ({ lang, setLang, t }), [lang, setLang, t])
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export function useI18n() {
  const ctx = useContext(Ctx)
  if (!ctx) throw new Error('useI18n outside provider')
  return ctx
}
