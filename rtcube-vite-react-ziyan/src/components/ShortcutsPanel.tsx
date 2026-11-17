import { useState } from 'react'
import { useI18n } from '@/i18n/I18nProvider'
import { emitInsertShortcut, loadShortcuts, saveShortcuts, type ShortcutItem } from '@/services/shortcuts'
import styles from './PlaceholderPanel.module.css'

type Props = {
  onPick?: (text: string) => void
}

export function ShortcutsPanel({ onPick }: Props) {
  const { t } = useI18n()
  const [list, setList] = useState<ShortcutItem[]>(() => loadShortcuts())
  const [draft, setDraft] = useState('')

  function add() {
    const text = draft.trim()
    if (!text) return
    const next = [...list, { id: `s_${Date.now()}`, text }]
    setList(next)
    saveShortcuts(next)
    setDraft('')
  }

  function remove(id: string) {
    const next = list.filter((x) => x.id !== id)
    setList(next)
    saveShortcuts(next)
  }

  return (
    <div className={styles.pane}>
      <h3>{t('shortcuts_title')}</h3>
      <p className={styles.desc}>
        {t('shortcuts_hint')}
        <br />
        点一条快捷语可填入当前聊天。
      </p>
      <div className={styles.row}>
        <input
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          placeholder="例如：好的，马上处理"
          onKeyDown={(e) => {
            if (e.key === 'Enter') add()
          }}
        />
        <button type="button" onClick={add}>
          {t('shortcuts_add')}
        </button>
      </div>
      <ul className={styles.list}>
        {list.map((item) => (
          <li key={item.id}>
            <button
              type="button"
              className={styles.useBtn}
              onClick={() => {
                emitInsertShortcut(item.text)
                onPick?.(item.text)
              }}
            >
              {item.text}
            </button>
            <button type="button" onClick={() => remove(item.id)}>
              删
            </button>
          </li>
        ))}
        {!list.length ? <li className={styles.empty}>{t('shortcuts_empty')}</li> : null}
      </ul>
    </div>
  )
}
