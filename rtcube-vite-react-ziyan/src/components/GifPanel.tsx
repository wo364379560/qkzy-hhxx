import { GIFT_LIST, type MediaAsset } from '@/im/stickerCatalog'
import styles from './GifPanel.module.css'

type Props = {
  onPick: (asset: MediaAsset) => void
}

export function GifPanel({ onPick }: Props) {
  return (
    <div className={styles.popup} onClick={(e) => e.stopPropagation()}>
      <div className={styles.grid}>
        {GIFT_LIST.map((g) => (
          <button key={g.id} type="button" className={styles.btn} title={g.name} onClick={() => onPick(g)}>
            <img src={g.url} alt={g.name} />
          </button>
        ))}
      </div>
      {!GIFT_LIST.length ? (
        <div className={styles.empty}>请把动图放到 src/assets/imagegif</div>
      ) : null}
    </div>
  )
}
