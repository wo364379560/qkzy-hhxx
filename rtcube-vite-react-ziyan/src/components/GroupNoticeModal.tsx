import { useEffect, useState } from 'react'
import { getGroupNotice, saveGroupNotice } from '@/api/groupNotice'
import { updateGroupDetail } from '@/im/groupActions'
import { useAppStore } from '@/store/AppStore'
import styles from './GroupNoticeModal.module.css'

const NOTICE_MAX = 600

type Props = {
  open: boolean
  groupId: string
  canEdit: boolean
  initial: string
  onClose: () => void
  onSaved: (content: string) => void
}

export function GroupNoticeModal({
  open,
  groupId,
  canEdit,
  initial,
  onClose,
  onSaved,
}: Props) {
  const { session } = useAppStore()
  const [text, setText] = useState(initial)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    if (!open) return
    setText(initial)
    let cancelled = false
    ;(async () => {
      try {
        const n = await getGroupNotice(groupId)
        if (!cancelled && n?.content) setText(n.content)
      } catch {
        /* 用 IM 公告 */
      }
    })()
    return () => {
      cancelled = true
    }
  }, [open, groupId, initial])

  if (!open) return null

  const nick = session?.nickname || session?.userId || ''
  const avatar = session?.avatar || ''
  const trimmed = text.trim()
  const canPublish = canEdit && trimmed.length >= 1 && trimmed.length <= NOTICE_MAX && !busy

  async function publish() {
    if (!canPublish) return
    setBusy(true)
    try {
      const content = trimmed.slice(0, NOTICE_MAX)
      try {
        await saveGroupNotice(groupId, content)
      } catch {
        /* 业务接口失败时仍同步 IM 公告 */
      }
      await updateGroupDetail(groupId, { notification: content.slice(0, 400) })
      onSaved(content)
      onClose()
    } catch (e) {
      alert(e instanceof Error ? e.message : '发布失败')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className={styles.mask} onClick={onClose} role="presentation">
      <div
        className={styles.modal}
        role="dialog"
        aria-labelledby="group-notice-title"
        onClick={(e) => e.stopPropagation()}
      >
        <header className={styles.head}>
          <h3 id="group-notice-title">{canEdit ? '发布新公告' : '群公告'}</h3>
        </header>
        {canEdit ? (
          <textarea
            className={styles.area}
            value={text}
            maxLength={NOTICE_MAX}
            placeholder={`填写公告，1-${NOTICE_MAX}字`}
            onChange={(e) => setText(e.target.value)}
          />
        ) : (
          <div className={styles.view}>{trimmed || '暂无公告'}</div>
        )}
        <footer className={styles.foot}>
          <div className={styles.author}>
            <span className={styles.avatar}>
              {avatar ? <img src={avatar} alt="" /> : nick.slice(0, 1)}
            </span>
            <span>{nick}</span>
          </div>
          <div className={styles.actions}>
            <button type="button" className={styles.cancel} onClick={onClose}>
              {canEdit ? '取消' : '关闭'}
            </button>
            {canEdit ? (
              <button
                type="button"
                className={styles.publish}
                disabled={!canPublish}
                onClick={() => void publish()}
              >
                {busy ? '发布中' : '发布'}
              </button>
            ) : null}
          </div>
        </footer>
      </div>
    </div>
  )
}
