import { useEffect, useRef, useState } from 'react'
import { getGroupNotice } from '@/api/groupNotice'
import { uploadAvatarFile } from '@/api/media'
import {
  INVITE_OPTION_LABELS,
  JOIN_OPTION_LABELS,
  SPEAK_FREQ_OPTIONS,
  canInviteGroupMembers,
  isGroupManagerRole,
  type JoinInviteOption,
} from '@/constants/groupIM'
import {
  dismissGroup,
  fetchGroupDetail,
  quitGroup,
  setBlockMemberProfile,
  setSpeakFreqSec,
  updateGroupDetail,
  type GroupDetail,
} from '@/im/groupActions'
import { applyMemberLivePatch, onGroupLiveChange } from '@/im/groupLive'
import { addGroupMembers, kickGroupMembers } from '@/im/groupMemberActions'
import { fetchGroupMemberPage, getChat } from '@/im/ImClient'
import type { GroupMember } from '@/im/types'
import {
  getConversationProfile,
  pinConversation,
  setConversationMute,
} from '@/im/profileActions'
import { useAppStore } from '@/store/AppStore'
import { useDetailUi } from '@/store/DetailUi'
import {
  GroupMemberContextMenu,
  type MemberMenuState,
} from './GroupMemberContextMenu'
import { GroupMemberAllModal } from './GroupMemberAllModal'
import { GroupMemberPickModal } from './GroupMemberPickModal'
import { GroupNoticeModal } from './GroupNoticeModal'
import { StickyHeroDrawer } from './StickyHeroDrawer'
import styles from './DetailPanels.module.css'

type Props = {
  open: boolean
  groupId: string
  onClose: () => void
}

export function GroupDetailDrawer({ open, groupId, onClose }: Props) {
  const { session, selectConversation } = useAppStore()
  const { openUserProfile } = useDetailUi()
  const [detail, setDetail] = useState<GroupDetail | null>(null)
  const [members, setMembers] = useState<GroupMember[]>([])
  const [pinned, setPinned] = useState(false)
  const [muted, setMuted] = useState(false)
  const [notice, setNotice] = useState('')
  const [noticeOpen, setNoticeOpen] = useState(false)
  const [joinOption, setJoinOption] = useState<JoinInviteOption>('NeedPermission')
  const [inviteOption, setInviteOption] = useState<JoinInviteOption>('FreeAccess')
  const [busy, setBusy] = useState('')
  const [canEditNotice, setCanEditNotice] = useState(false)
  const [nameOpen, setNameOpen] = useState(false)
  const [nameDraft, setNameDraft] = useState('')
  const [pickMode, setPickMode] = useState<'add' | 'remove' | null>(null)
  const [allOpen, setAllOpen] = useState(false)
  const [menu, setMenu] = useState<MemberMenuState>(null)
  const fileRef = useRef<HTMLInputElement>(null)

  const convId = `GROUP${groupId}`
  const selfId = session?.userId || ''
  const isOwner = Boolean(detail && selfId && detail.ownerID === selfId)
  const canManage = isOwner || canEditNotice

  useEffect(() => {
    if (!open || !groupId) return
    setNoticeOpen(false)
    setNameOpen(false)
    setPickMode(null)
    setAllOpen(false)
    setMenu(null)
    let cancelled = false
    ;(async () => {
      try {
        const d = await fetchGroupDetail(groupId)
        if (cancelled) return
        setDetail(d)
        try {
          const page = await fetchGroupMemberPage(groupId, 0, 20)
          if (!cancelled) setMembers(page.members)
        } catch {
          if (!cancelled) setMembers([])
        }
        setJoinOption(d.joinOption)
        setInviteOption(d.inviteOption)
        try {
          const n = await getGroupNotice(groupId)
          if (!cancelled) setNotice(n?.content || d.notification || '')
        } catch {
          if (!cancelled) setNotice(d.notification || '')
        }
        const conv = await getConversationProfile(convId)
        if (!cancelled && conv) {
          setPinned(Boolean(conv.isPinned))
          setMuted(
            conv.messageRemindType === 'AcceptNotNotify' ||
              conv.messageRemindType === 'MSG_REMIND_ACPT_NOT_NOTE',
          )
        }
        let edit = Boolean(selfId && d.ownerID === selfId)
        if (selfId && !edit) {
          try {
            const chat = getChat()
            const res = await chat.getGroupMemberProfile({
              groupID: groupId,
              userIDList: [selfId],
            })
            const role = String(res?.data?.memberList?.[0]?.role || '')
            edit = isGroupManagerRole(role)
          } catch {
            /* ignore */
          }
        }
        if (!cancelled) setCanEditNotice(edit)
      } catch (e) {
        console.error(e)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [open, groupId, convId, selfId])

  useEffect(() => {
    if (!open || !groupId) return
    return onGroupLiveChange((e) => {
      if (e.groupID !== groupId) return
      if (e.added?.length || e.removedIds?.length || e.updated?.length) {
        setMembers((prev) => {
          const { next, addedCount, removedCount } = applyMemberLivePatch(prev, e)
          if (addedCount || removedCount) {
            setDetail((d) =>
              d
                ? {
                    ...d,
                    memberCount: Math.max(0, Number(d.memberCount || 0) + addedCount - removedCount),
                  }
                : d,
            )
          }
          return next
        })
        return
      }
    })
  }, [open, groupId])

  async function run(label: string, fn: () => Promise<void>) {
    setBusy(label)
    try {
      await fn()
    } catch (e) {
      alert(e instanceof Error ? e.message : `${label}失败`)
    } finally {
      setBusy('')
    }
  }

  const noticeText = notice.trim() || detail?.notification?.trim() || ''
  const memberCount = detail?.memberCount || members.length
  const previewMembers = members.slice(0, 8)
  const canInvite = canInviteGroupMembers(inviteOption, canManage)
  const canRemove = canManage

  async function reloadMembers() {
    try {
      const [d, page] = await Promise.all([
        fetchGroupDetail(groupId),
        fetchGroupMemberPage(groupId, 0, 20),
      ])
      setDetail(d)
      setMembers(page.members)
    } catch {
      /* ignore */
    }
  }

  async function onPickAvatar(file?: File) {
    if (!file || !canManage) return
    try {
      const url = await uploadAvatarFile(file, { type: 'group', groupId })
      await updateGroupDetail(groupId, { avatar: url })
      setDetail((d) => (d ? { ...d, avatar: url } : d))
    } catch (e) {
      alert(e instanceof Error ? e.message : '更换头像失败')
    }
  }

  async function saveName() {
    const next = nameDraft.trim()
    if (!next) return
    await run('改名', async () => {
      await updateGroupDetail(groupId, { name: next })
      setDetail((d) => (d ? { ...d, name: next } : d))
      setNameOpen(false)
    })
  }

  return (
    <>
      <input
        ref={fileRef}
        type="file"
        accept="image/*"
        hidden
        onChange={(e) => {
          const file = e.target.files?.[0]
          e.target.value = ''
          void onPickAvatar(file)
        }}
      />
      <StickyHeroDrawer
        open={open}
        theme="light"
        title={detail?.name || '群详情'}
        subtitle={detail ? `${detail.memberCount} 人 · ID ${groupId}` : groupId}
        avatarUrl={detail?.avatar}
        onClose={onClose}
        showCamera={canManage}
        onAvatarClick={canManage ? () => fileRef.current?.click() : undefined}
        onTitleClick={
          canManage
            ? () => {
                setNameDraft(detail?.name || '')
                setNameOpen(true)
              }
            : undefined
        }
      >
        <div className={styles.section}>
          <div className={styles.sectionLabel}>群公告</div>
          <button type="button" className={styles.cellBtn} onClick={() => setNoticeOpen(true)}>
            <span className={noticeText ? undefined : styles.cellMuted}>{noticeText || '暂无公告'}</span>
            <em>›</em>
          </button>
        </div>

        <div className={styles.card}>
          <button type="button" className={styles.memberHead} onClick={() => setAllOpen(true)}>
            <strong>群聊成员</strong>
            <span>查看{memberCount}名群成员 ›</span>
          </button>
          <div className={styles.memberGrid}>
            {previewMembers.map((m) => (
              <button
                key={m.userID}
                type="button"
                className={styles.memberCell}
                title={m.nick || m.userID}
                onClick={() =>
                  openUserProfile({
                    userId: m.userID,
                    nickname: m.nick,
                    avatar: m.avatar,
                  })
                }
                onContextMenu={(e) => {
                  e.preventDefault()
                  setMenu({ x: e.clientX, y: e.clientY, member: m })
                }}
              >
                <b>
                  {m.avatar ? <img src={m.avatar} alt="" /> : (m.nick || m.userID).slice(0, 1)}
                </b>
                <em>{m.nick || m.userID}</em>
              </button>
            ))}
            {canInvite ? (
              <button
                type="button"
                className={`${styles.memberCell} ${styles.memberAct}`}
                onClick={() => setPickMode('add')}
              >
                <b>+</b>
                <em>邀请</em>
              </button>
            ) : null}
            {canRemove ? (
              <button
                type="button"
                className={`${styles.memberCell} ${styles.memberAct}`}
                onClick={() => setPickMode('remove')}
              >
                <b>−</b>
                <em>移出</em>
              </button>
            ) : null}
          </div>
        </div>

        <div className={styles.card}>
          <div className={styles.cardTitle}>会话设置</div>
          <label className={styles.switchRow}>
            <span>置顶聊天</span>
            <input
              type="checkbox"
              checked={pinned}
              disabled={Boolean(busy)}
              onChange={(e) => {
                const v = e.target.checked
                setPinned(v)
                void run('置顶', () => pinConversation(convId, v))
              }}
            />
          </label>
          <label className={styles.switchRow}>
            <span>消息免打扰</span>
            <input
              type="checkbox"
              checked={muted}
              disabled={Boolean(busy)}
              onChange={(e) => {
                const v = e.target.checked
                setMuted(v)
                void run('免打扰', () => setConversationMute(convId, v))
              }}
            />
          </label>
        </div>

        {isOwner ? (
          <div className={styles.card}>
            <div className={styles.cardTitle}>群管理</div>
            <label className={styles.row}>
              <span>加群方式</span>
              <select
                value={joinOption}
                onChange={(e) => setJoinOption(e.target.value as JoinInviteOption)}
              >
                {Object.entries(JOIN_OPTION_LABELS).map(([v, label]) => (
                  <option key={v} value={v}>
                    {label}
                  </option>
                ))}
              </select>
            </label>
            <label className={styles.row}>
              <span>邀请方式</span>
              <select
                value={inviteOption}
                onChange={(e) => setInviteOption(e.target.value as JoinInviteOption)}
              >
                {Object.entries(INVITE_OPTION_LABELS).map(([v, label]) => (
                  <option key={v} value={v}>
                    {label}
                  </option>
                ))}
              </select>
            </label>
            <button
              type="button"
              className={styles.secondaryBtn}
              disabled={Boolean(busy)}
              onClick={() =>
                void run('保存验证', async () => {
                  await updateGroupDetail(groupId, { joinOption, inviteOption })
                })
              }
            >
              保存加群验证
            </button>
            <label className={styles.switchRow}>
              <span>禁止成员查看资料</span>
              <input
                type="checkbox"
                checked={Boolean(detail?.blockMemberProfile)}
                disabled={Boolean(busy)}
                onChange={(e) => {
                  const v = e.target.checked
                  setDetail((d) => (d ? { ...d, blockMemberProfile: v } : d))
                  void run('权限', () => setBlockMemberProfile(groupId, v))
                }}
              />
            </label>
            <p className={styles.hint}>开启后普通成员不可互看详情；群主/管理员不受限。</p>
          </div>
        ) : null}

        {canManage ? (
          <div className={styles.card}>
            <label className={styles.switchRow}>
              <span>全员禁言</span>
              <input
                type="checkbox"
                className={styles.iosSwitch}
                checked={Boolean(detail?.muteAllMembers)}
                disabled={Boolean(busy)}
                onChange={(e) => {
                  const v = e.target.checked
                  setDetail((d) => (d ? { ...d, muteAllMembers: v } : d))
                  void run('全员禁言', () => updateGroupDetail(groupId, { muteAllMembers: v }))
                }}
              />
            </label>
            <label className={styles.row}>
              <span>发言限频</span>
              <select
                className={styles.ghostSelect}
                value={detail?.speakFreqSec ?? 0}
                disabled={Boolean(busy)}
                onChange={(e) => {
                  const sec = Number(e.target.value) || 0
                  setDetail((d) => (d ? { ...d, speakFreqSec: sec } : d))
                  void run('发言限频', () => setSpeakFreqSec(groupId, sec))
                }}
              >
                {SPEAK_FREQ_OPTIONS.map((opt) => (
                  <option key={opt.sec} value={opt.sec}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </label>
          </div>
        ) : null}

        <div className={styles.card}>
          <button
            type="button"
            className={styles.dangerRow}
            disabled={Boolean(busy)}
            onClick={() => {
              if (isOwner) {
                if (!confirm('解散群组？此操作不可恢复')) return
                void run('解散', async () => {
                  await dismissGroup(groupId)
                  onClose()
                  void selectConversation('')
                })
              } else {
                if (!confirm('退出该群？')) return
                void run('退群', async () => {
                  await quitGroup(groupId)
                  onClose()
                  void selectConversation('')
                })
              }
            }}
          >
            {isOwner ? '解散群组' : '退出群聊'}
          </button>
        </div>
      </StickyHeroDrawer>

      <GroupNoticeModal
        open={open && noticeOpen}
        groupId={groupId}
        canEdit={canManage}
        initial={noticeText}
        onClose={() => setNoticeOpen(false)}
        onSaved={(content) => {
          setNotice(content)
          setDetail((d) => (d ? { ...d, notification: content.slice(0, 400) } : d))
        }}
      />
      <GroupMemberPickModal
        open={Boolean(pickMode)}
        mode={pickMode || 'add'}
        groupId={groupId}
        members={members}
        excludeIds={
          pickMode === 'remove'
            ? [selfId, detail?.ownerID || '']
            : [selfId, ...members.map((m) => m.userID)]
        }
        onClose={() => setPickMode(null)}
        onConfirm={async (userIDs) => {
          if (pickMode === 'add') {
            await addGroupMembers(groupId, userIDs)
          } else {
            await kickGroupMembers(groupId, userIDs)
          }
          await reloadMembers()
        }}
      />
      <GroupMemberAllModal
        open={allOpen}
        groupId={groupId}
        selfId={selfId}
        selfRole={isOwner ? 'Owner' : canManage ? 'Admin' : 'Member'}
        onClose={() => setAllOpen(false)}
      />
      <GroupMemberContextMenu
        menu={menu}
        groupId={groupId}
        selfId={selfId}
        selfRole={isOwner ? 'Owner' : canManage ? 'Admin' : 'Member'}
        onClose={() => setMenu(null)}
        onChanged={() => void reloadMembers()}
      />
      {nameOpen ? (
        <div className={styles.nameMask} onClick={() => setNameOpen(false)} role="presentation">
          <div className={styles.nameBox} onClick={(e) => e.stopPropagation()}>
            <h3>修改群名称</h3>
            <input
              value={nameDraft}
              maxLength={30}
              onChange={(e) => setNameDraft(e.target.value)}
            />
            <div className={styles.nameActs}>
              <button type="button" className={styles.nameCancel} onClick={() => setNameOpen(false)}>
                取消
              </button>
              <button type="button" className={styles.nameOk} onClick={() => void saveName()}>
                确定
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </>
  )
}
