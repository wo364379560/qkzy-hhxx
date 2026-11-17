import { useRef, useState, type ReactNode, type UIEvent } from 'react'
import styles from './StickyHeroDrawer.module.css'

type Props = {
  open: boolean
  title: string
  subtitle?: string
  avatarUrl?: string
  theme?: 'dark' | 'light'
  avatarExtra?: ReactNode
  headerRight?: ReactNode
  showCamera?: boolean
  onAvatarClick?: () => void
  onTitleClick?: () => void
  onClose: () => void
  children: ReactNode
  footer?: ReactNode
}

/**
 * 对照「2026年5月27日-Android」contact_detail / group_detail 吸顶：
 * 展开时头像铺满 hero；上滑后收成 sticky 顶栏（小头像+标题）。
 */
export function StickyHeroDrawer({
  open,
  title,
  subtitle,
  avatarUrl,
  theme = 'dark',
  avatarExtra,
  headerRight,
  showCamera,
  onAvatarClick,
  onTitleClick,
  onClose,
  children,
  footer,
}: Props) {
  const scrollerRef = useRef<HTMLDivElement>(null)
  const [progress, setProgress] = useState(0) // 0 expanded → 1 collapsed
  const light = theme === 'light'

  if (!open) return null

  function onScroll(e: UIEvent<HTMLDivElement>) {
    const el = e.currentTarget
    const range = 140
    const p = Math.min(1, Math.max(0, el.scrollTop / range))
    setProgress(p)
  }

  const heroScale = 1 + (1 - progress) * 0.08
  const heroOp = 1 - progress * 0.35
  const barOp = progress
  const miniAvOp = Math.min(1, Math.max(0, (progress - 0.35) / 0.4))

  return (
    <>
      <button type="button" className={styles.backdrop} aria-label="关闭" onClick={onClose} />
      <aside className={styles.drawer} role="dialog" aria-modal>
        <div
          className={`${styles.stickyBar} ${headerRight ? styles.stickyBarWide : ''}`}
          style={{ opacity: Math.max(barOp, light ? 1 : 0.15) }}
        >
          <button type="button" className={styles.back} onClick={onClose}>
            ‹
          </button>
          <div className={styles.barCenter} style={{ opacity: miniAvOp }}>
            <div className={styles.miniAvatar}>
              {avatarUrl ? <img src={avatarUrl} alt="" /> : title.slice(0, 1)}
            </div>
            <div className={styles.barText}>
              <strong>{title}</strong>
              {subtitle ? <span>{subtitle}</span> : null}
            </div>
          </div>
          {headerRight ? <div className={styles.barRight}>{headerRight}</div> : null}
          <button type="button" className={styles.close} onClick={onClose}>
            ✕
          </button>
        </div>

        <div className={styles.scroller} ref={scrollerRef} onScroll={onScroll}>
          <div
            className={`${styles.hero} ${light ? styles.heroLight : ''}`}
            style={{ opacity: heroOp }}
          >
            {light ? null : (
              <>
                <div
                  className={styles.heroBg}
                  style={{
                    transform: `scale(${heroScale})`,
                    backgroundImage: avatarUrl ? `url(${avatarUrl})` : undefined,
                  }}
                />
                <div className={styles.heroShade} />
              </>
            )}
            <div className={styles.heroContent} style={{ opacity: 1 - progress * 1.2 }}>
              <div className={styles.avatarWrap}>
                <button
                  type="button"
                  className={styles.heroAvatarBtn}
                  disabled={!onAvatarClick}
                  onClick={onAvatarClick}
                >
                  <div className={styles.heroAvatar}>
                    {avatarUrl ? <img src={avatarUrl} alt="" /> : title.slice(0, 1)}
                  </div>
                  {showCamera ? <span className={styles.cameraBar}>📷</span> : null}
                </button>
                {avatarExtra}
              </div>
              {onTitleClick ? (
                <button type="button" className={styles.titleBtn} onClick={onTitleClick}>
                  <h2>{title}</h2>
                </button>
              ) : (
                <h2>{title}</h2>
              )}
              {subtitle ? <p>{subtitle}</p> : null}
              {headerRight ? <div className={styles.heroRight}>{headerRight}</div> : null}
            </div>
          </div>

          <div className={styles.body}>{children}</div>
          {footer ? <div className={styles.footer}>{footer}</div> : null}
          <div className={styles.pad} />
        </div>
      </aside>
    </>
  )
}
