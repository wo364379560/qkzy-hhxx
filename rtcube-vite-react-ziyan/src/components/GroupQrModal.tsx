import { useMemo, useState } from 'react'
import iconCopy from '@/assets/icon/copy.svg'
import iconForward from '@/assets/icon/forward.svg'
import styles from './GroupQrModal.module.css'

type Props = {
  open: boolean
  groupId: string
  name: string
  avatarUrl?: string
  onClose: () => void
}

function qrSrc(data: string, size = 280) {
  return `https://api.qrserver.com/v1/create-qr-code/?size=${size}x${size}&margin=8&data=${encodeURIComponent(data)}`
}

export function GroupQrModal({ open, groupId, name, avatarUrl, onClose }: Props) {
  const [tip, setTip] = useState('')
  const joinUrl = useMemo(() => `chengnan://group/${groupId}`, [groupId])
  const img = useMemo(() => qrSrc(joinUrl), [joinUrl])

  if (!open) return null

  async function copy(text: string, ok: string) {
    try {
      await navigator.clipboard.writeText(text)
      setTip(ok)
    } catch {
      setTip('复制失败')
    }
    window.setTimeout(() => setTip(''), 1600)
  }

  async function saveImage() {
    try {
      const res = await fetch(img)
      const blob = await res.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${name || 'group'}-qr.png`
      a.click()
      URL.revokeObjectURL(url)
      setTip('已保存')
    } catch {
      window.open(img, '_blank')
    }
    window.setTimeout(() => setTip(''), 1600)
  }

  return (
    <div className={styles.mask} onClick={onClose} role="presentation">
      <div className={styles.wrap} onClick={(e) => e.stopPropagation()}>
        <div className={styles.card}>
          <div className={styles.top}>
            <div className={styles.avatar}>
              {avatarUrl ? <img src={avatarUrl} alt="" /> : name.slice(0, 1)}
            </div>
            <div className={styles.meta}>
              <strong>{name || '群聊'}</strong>
              <p>
                群号: {groupId}
                <button
                  type="button"
                  className={styles.copyId}
                  title="复制群号"
                  onClick={() => void copy(groupId, '已复制群号')}
                >
                  <img src={iconCopy} alt="" />
                </button>
              </p>
            </div>
          </div>
          <div className={styles.dash} />
          <img className={styles.qr} src={img} alt="群二维码" />
        </div>
        <p className={styles.hint}>扫一扫二维码，加入群聊</p>
        {tip ? <p className={styles.tip}>{tip}</p> : null}
        <div className={styles.actions}>
          <button type="button" onClick={() => void copy(joinUrl, '已复制，可转发')}>
            <img src={iconForward} alt="" />
            转发
          </button>
          <button type="button" onClick={() => void copy(joinUrl, '已复制链接')}>
            <img src={iconCopy} alt="" />
            复制链接
          </button>
          <button type="button" onClick={() => void saveImage()}>
            <span className={styles.saveIco} />
            保存图片
          </button>
        </div>
        <button type="button" className={styles.close} onClick={onClose}>
          关闭
        </button>
      </div>
    </div>
  )
}
