import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import { submitReport } from '@/api/report'
import { emitAtMember } from '@/im/atMemberEvent'
import {
  kickGroupMembers,
  setGroupMemberMute,
  setGroupMemberNameCard,
  setGroupMemberRole,
} from '@/im/groupMemberActions'
import type { GroupMember } from '@/im/types'
import { useDetailUi } from '@/store/DetailUi'
import styles from './GroupMemberContextMenu.module.css'

export type MemberMenuState = {
  x: number
  y: number
  member: GroupMember
} | null

const MUTE_PRESETS = [
  { label: '10分钟', sec: 10 * 60 },
  { label: '1小时', sec: 60 * 60 },
  { label: '12小时', sec: 12 * 60 * 60 },
  { label: '1天', sec: 24 * 60 * 60 },
  { label: '30天', sec: 30 * 24 * 60 * 60 },
  { label: '解除禁言', sec: 0 },
]

type Props = {
  menu: MemberMenuState
  groupId: string
  selfId: string
  selfRole: 'Owner' | 'Admin' | 'Member'
  restricted?: boolean
  onClose: () => void
  onChanged?: () => void
}

export function GroupMemberContextMenu({
  menu,
  groupId,
  selfId,
  selfRole,
  restricted = false,
  onClose,
  onChanged,
}: Props) {
  const { openUserProfile, navigateToChat } = useDetailUi()
  const [subOpen, setSubOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)
  const subWrapRef = useRef<HTMLDivElement>(null)
  const subRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    setSubOpen(false)
  }, [menu])

  useEffect(() => {
    if (!menu) return
    const close = (event: Event) => {
      if (event.target instanceof Node && menuRef.current?.contains(event.target)) return
      onClose()
    }
    const t = window.setTimeout(() => {
      window.addEventListener('click', close)
      window.addEventListener('scroll', close, true)
    }, 0)
    return () => {
      window.clearTimeout(t)
      window.removeEventListener('click', close)
      window.removeEventListener('scroll', close, true)
    }
  }, [menu, onClose])

  useLayoutEffect(() => {
    if (!menu || !menuRef.current) return
    const menuElement = menuRef.current
    const placeMenu = () => {
      const gap = 4
      const margin = 8
      const clamp = (value: number, size: number, viewport: number) =>
        Math.max(margin, Math.min(value, viewport - size - margin))
      const subElement = subRef.current
      // Reserve room on the left when opening the mute presets.
      const preferredLeft = subElement
        ? Math.max(menu.x, subElement.offsetWidth + gap + margin)
        : menu.x
      const left = clamp(preferredLeft, menuElement.offsetWidth, window.innerWidth)
      const top = clamp(menu.y, menuElement.offsetHeight, window.innerHeight)
      menuElement.style.left = `${left}px`
      menuElement.style.top = `${top}px`
      if (subElement && subWrapRef.current) {
        const anchor = subWrapRef.current.getBoundingClientRect()
        subElement.style.left = `${clamp(left - subElement.offsetWidth - gap, subElement.offsetWidth, window.innerWidth)}px`
        subElement.style.top = `${clamp(anchor.bottom - subElement.offsetHeight, subElement.offsetHeight, window.innerHeight)}px`
      }
    }
    placeMenu()
    window.addEventListener('resize', placeMenu)
    menuElement.addEventListener('scroll', placeMenu)
    return () => {
      window.removeEventListener('resize', placeMenu)
      menuElement.removeEventListener('scroll', placeMenu)
    }
  }, [menu, subOpen, restricted, selfId, selfRole])

  if (!menu) return null

  const m = menu.member
  const isSelf = m.userID === selfId
  const isOwner = selfRole === 'Owner'
  const isManager = selfRole === 'Owner' || selfRole === 'Admin'
  const targetOwner = m.role === 'Owner'
  const canKick = isManager && !isSelf && !targetOwner && !(selfRole === 'Admin' && m.role === 'Admin')
  const canMute = isManager && !isSelf && !targetOwner
  const canSetAdmin = isOwner && !isSelf && !targetOwner
  const canEditCard = isSelf || (isManager && !targetOwner) || isOwner

  async function run(fn: () => Promise<void>) {
    if (busy) return
    setBusy(true)
    try {
      await fn()
      onChanged?.()
      onClose()
    } catch (e) {
      alert(e instanceof Error ? e.message : '操作失败')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div
      ref={menuRef}
      className={styles.menu}
      style={{ left: menu.x, top: menu.y }}
      onClick={(e) => e.stopPropagation()}
      onContextMenu={(e) => e.preventDefault()}
    >
      {!isSelf && !restricted ? (
        <button
          type="button"
          onClick={() => {
            onClose()
            navigateToChat(`C2C${m.userID}`)
          }}
        >
          <span>发送消息</span>
          <i className={styles.icBubble} />
        </button>
      ) : null}
      {!isSelf ? (
        <button
          type="button"
          onClick={() => {
            emitAtMember({ groupId, userID: m.userID, nick: m.nick || m.userID })
            onClose()
            navigateToChat(`GROUP${groupId}`)
          }}
        >
          <span>@ TA</span>
          <i className={styles.icAt} />
        </button>
      ) : null}
      <button
        type="button"
        onClick={() => {
          onClose()
          openUserProfile({
            userId: m.userID,
            nickname: m.nick,
            avatar: m.avatar,
            restricted,
          })
        }}
      >
        <span>查看资料</span>
        <i className={styles.icCard} />
      </button>
      {canSetAdmin ? (
        <button
          type="button"
          disabled={busy}
          onClick={() =>
            void run(async () => {
              const next = m.role === 'Admin' ? 'Member' : 'Admin'
              await setGroupMemberRole(groupId, m.userID, next)
            })
          }
        >
          <span>{m.role === 'Admin' ? '取消管理员' : '设为管理员'}</span>
          <i className={styles.icUser} />
        </button>
      ) : null}
      {canEditCard ? (
        <button
          type="button"
          disabled={busy}
          onClick={() => {
            const next = window.prompt('修改群昵称', m.nick || '')
            if (next == null) return
            void run(async () => {
              await setGroupMemberNameCard(groupId, m.userID, next.trim())
            })
          }}
        >
          <span>修改群昵称</span>
          <i className={styles.icEdit} />
        </button>
      ) : null}

      {canKick || !isSelf || canMute ? <div className={styles.hr} /> : null}

      {canKick ? (
        <button
          type="button"
          className={styles.danger}
          disabled={busy}
          onClick={() => {
            if (!confirm(`将 ${m.nick || m.userID} 移出本群？`)) return
            void run(async () => {
              await kickGroupMembers(groupId, [m.userID])
            })
          }}
        >
          <span>移出本群</span>
          <i className={styles.icKick} />
        </button>
      ) : null}
      {!isSelf ? (
        <button
          type="button"
          disabled={busy}
          onClick={() =>
            void run(async () => {
              await submitReport({
                targetId: m.userID,
                targetType: 'user',
                targetName: m.nick || m.userID,
                category: 'other',
                description: `群 ${groupId} 内举报`,
              })
              alert('已提交举报')
            })
          }
        >
          <span>举报</span>
          <i className={styles.icWarn} />
        </button>
      ) : null}
      {canMute ? (
        <button
          type="button"
          disabled={busy}
          onClick={() =>
            void run(async () => {
              await setGroupMemberMute(groupId, m.userID, 30 * 24 * 60 * 60)
            })
          }
        >
          <span>屏蔽此人发言</span>
          <i className={styles.icMute} />
        </button>
      ) : null}
      {canMute ? (
        <div ref={subWrapRef} className={styles.subWrap}>
          <button
            type="button"
            className={styles.hasSub}
            aria-expanded={subOpen}
            onClick={() => setSubOpen((v) => !v)}
          >
            <span>设置群内禁言</span>
            <i className={styles.icChevron} />
          </button>
          {subOpen ? (
            <div ref={subRef} className={styles.sub}>
              {MUTE_PRESETS.map((p) => (
                <button
                  key={p.label}
                  type="button"
                  disabled={busy}
                  onClick={() =>
                    void run(async () => {
                      await setGroupMemberMute(groupId, m.userID, p.sec)
                    })
                  }
                >
                  {p.label}
                </button>
              ))}
            </div>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}
