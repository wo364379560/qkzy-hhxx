/**
 * 个人详情 — 对照「2026年5月27日-Android」ContactDetailPage + web-h5 UserDetailView
 */
import { useEffect, useState } from 'react'
import { addFriend, checkIsFriend, formatFriendAddError } from '@/im/friends'
import {
  addToBlacklist,
  deleteFriend,
  fetchUserProfile,
  getBlacklist,
  getConversationProfile,
  pinConversation,
  removeFromBlacklist,
  setConversationMute,
  updateFriendRemark,
} from '@/im/profileActions'
import { useAppStore } from '@/store/AppStore'
import { useCallSession } from '@/store/CallSession'
import { useDetailUi } from '@/store/DetailUi'
import { StickyHeroDrawer } from './StickyHeroDrawer'
import styles from './DetailPanels.module.css'

type Props = {
  open: boolean
  userId: string
  nickname?: string
  avatar?: string
  restricted?: boolean
  onClose: () => void
}

export function UserProfileDrawer({
  open,
  userId,
  nickname,
  avatar,
  restricted,
  onClose,
}: Props) {
  const { session } = useAppStore()
  const { navigateToChat } = useDetailUi()
  const call = useCallSession()
  const [nick, setNick] = useState(nickname || userId)
  const [face, setFace] = useState(avatar || '')
  const [signature, setSignature] = useState('')
  const [remark, setRemark] = useState('')
  const [pinned, setPinned] = useState(false)
  const [muted, setMuted] = useState(false)
  const [blocked, setBlocked] = useState(false)
  const [isFriend, setIsFriend] = useState(false)
  const [addWording, setAddWording] = useState('你好，我想加你为好友')
  const [busy, setBusy] = useState('')
  const [copied, setCopied] = useState(false)

  const self = session?.userId === userId
  const convId = `C2C${userId}`
  const displayName = remark || nick || userId

  useEffect(() => {
    if (!open || !userId) return
    setNick(nickname || userId)
    setFace(avatar || '')
    setSignature('')
    setIsFriend(false)
    setRemark('')
    let cancelled = false
    ;(async () => {
      try {
        if (!restricted) {
          const [p, friend, bl] = await Promise.all([
            fetchUserProfile(userId),
            checkIsFriend(userId).catch(() => false),
            getBlacklist().catch(() => [] as string[]),
          ])
          if (cancelled) return
          setNick(p.nick)
          setFace(p.avatar)
          setSignature(p.selfSignature || '')
          setIsFriend(friend)
          setBlocked(bl.includes(userId))

          if (friend) {
            const conv = await getConversationProfile(convId).catch(() => null)
            if (cancelled) return
            if (conv) {
              setPinned(Boolean(conv.isPinned))
              setMuted(
                conv.messageRemindType === 'AcceptNotNotify' ||
                  conv.messageRemindType === 'MSG_REMIND_ACPT_NOT_NOTE',
              )
              if (conv.remark) setRemark(String(conv.remark))
            }
          }
        }
      } catch (e) {
        console.error(e)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [open, userId, nickname, avatar, restricted, convId])

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

  function copyId() {
    void navigator.clipboard?.writeText(userId).then(() => {
      setCopied(true)
      window.setTimeout(() => setCopied(false), 1500)
    })
  }

  function openChat() {
    navigateToChat(convId)
  }

  return (
    <StickyHeroDrawer
      open={open}
      title={restricted ? '个人详情' : displayName}
      subtitle={restricted ? '对方设置了隐私' : `ID ${userId}`}
      avatarUrl={restricted ? '' : face}
      onClose={onClose}
      footer={
        !self && !restricted && isFriend ? (
          <button type="button" className={styles.primaryBtn} onClick={openChat}>
            发消息
          </button>
        ) : null
      }
    >
      {restricted ? (
        <div className={styles.card}>
          <p className={styles.hint}>群已禁止普通成员互看资料，仅可查看群主/管理员。</p>
        </div>
      ) : (
        <>
          {!self ? (
            <div className={styles.actions}>
              {isFriend ? (
                <>
                  <button type="button" className={styles.action} onClick={openChat}>
                    <span>💬</span>
                    消息
                  </button>
                  <button
                    type="button"
                    className={styles.action}
                    onClick={() => {
                      onClose()
                      void call.startCall({
                        peerId: userId,
                        peerName: displayName,
                        mediaType: 1,
                      })
                    }}
                    title="语音通话"
                  >
                    <span>📞</span>
                    语音
                  </button>
                  <button
                    type="button"
                    className={styles.action}
                    onClick={() => {
                      onClose()
                      void call.startCall({
                        peerId: userId,
                        peerName: displayName,
                        mediaType: 2,
                      })
                    }}
                    title="视频通话"
                  >
                    <span>📹</span>
                    视频
                  </button>
                  <button
                    type="button"
                    className={styles.action}
                    disabled={Boolean(busy)}
                    onClick={() => {
                      const v = !muted
                      setMuted(v)
                      void run('免打扰', () => setConversationMute(convId, v))
                    }}
                  >
                    <span>{muted ? '🔕' : '🔔'}</span>
                    {muted ? '已关闭' : '通知'}
                  </button>
                </>
              ) : (
                <div className={styles.addBox}>
                  <input
                    value={addWording}
                    onChange={(e) => setAddWording(e.target.value)}
                    placeholder="验证消息"
                  />
                  <button
                    type="button"
                    className={styles.primaryBtn}
                    disabled={Boolean(busy)}
                    onClick={() =>
                      void run('加好友', async () => {
                        try {
                          await addFriend(userId, addWording.trim() || '你好，我想加你为好友')
                          alert('好友申请已发送')
                        } catch (e) {
                          throw new Error(formatFriendAddError(e))
                        }
                      })
                    }
                  >
                    添加好友
                  </button>
                </div>
              )}
            </div>
          ) : null}

          <div className={styles.card}>
            <button type="button" className={styles.accountRow} onClick={copyId}>
              <div>
                <div className={styles.cardTitle}>账号</div>
                <strong>{userId}</strong>
              </div>
              <em>{copied ? '已复制' : '复制'}</em>
            </button>
          </div>

          {signature ? (
            <div className={styles.card}>
              <div className={styles.cardTitle}>个性签名</div>
              <p className={styles.sig}>{signature}</p>
            </div>
          ) : null}

          {self ? (
            <div className={styles.card}>
              <p className={styles.hint}>这是你自己的资料。</p>
            </div>
          ) : null}

          {!self && isFriend ? (
            <div className={styles.card}>
              <div className={styles.cardTitle}>好友设置</div>
              <label className={styles.row}>
                <span>设置备注名</span>
                <input
                  value={remark}
                  placeholder="添加备注"
                  onChange={(e) => setRemark(e.target.value)}
                  onBlur={() => {
                    void run('备注', () => updateFriendRemark(userId, remark.trim()))
                  }}
                />
              </label>
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
          ) : null}

          {!self ? (
            <div className={styles.card}>
              <button
                type="button"
                className={styles.dangerRow}
                disabled={Boolean(busy)}
                onClick={() =>
                  void run(blocked ? '取消拉黑' : '拉黑', async () => {
                    if (blocked) await removeFromBlacklist(userId)
                    else {
                      if (!confirm('确认加入黑名单？')) return
                      await addToBlacklist(userId)
                    }
                    setBlocked(!blocked)
                  })
                }
              >
                {blocked ? '取消拉黑' : '加入黑名单'}
              </button>
              {isFriend ? (
                <button
                  type="button"
                  className={styles.dangerRow}
                  disabled={Boolean(busy)}
                  onClick={() => {
                    if (!confirm('确定删除好友？')) return
                    void run('删除好友', async () => {
                      await deleteFriend(userId)
                      setIsFriend(false)
                      onClose()
                    })
                  }}
                >
                  删除好友
                </button>
              ) : null}
            </div>
          ) : null}
        </>
      )}
    </StickyHeroDrawer>
  )
}
