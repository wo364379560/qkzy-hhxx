import { EMOJI_ITEMS } from '@/im/emojiCatalog'
import styles from './EmojiPanel.module.css'

type Props = {
  onPick: (tag: string) => void
}

export function EmojiPanel({ onPick }: Props) {
  return (
    <div className={styles.popup} onClick={(e) => e.stopPropagation()}>
      <div className={styles.grid}>
        {EMOJI_ITEMS.map((e) => (
          <button key={e.tag} type="button" className={styles.btn} onClick={() => onPick(e.tag)}>
            {e.assetUrl ? <img src={e.assetUrl} alt="" /> : <span>{e.glyph}</span>}
          </button>
        ))}
      </div>
    </div>
  )
}
