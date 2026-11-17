import { useEffect, useMemo, useState } from 'react'
import { DEFAULT_GROUP_KIND, GROUP_KIND_LABELS } from '@/constants/groupIM'
import { createGroupWithTip, type CreateGroupKind } from '@/im/groupCreate'
import { fetchFriendList, type FriendRow } from '@/im/ImClient'
import { useAppStore } from '@/store/AppStore'
import {
  generateCommunityGroupId,
  generatePublicGroupId,
} from '@/utils/groupId'
import styles from './CreateGroupModal.module.css'

type Props = {
  open: boolean
  onClose: () => void
  onSuccess?: (conversationID: string) => void
}

type Step = 'pick' | 'type'

export function CreateGroupModal({ open, onClose, onSuccess }: Props) {
  const { session } = useAppStore()
  const [step, setStep] = useState<Step>('pick')
  const [friends, setFriends] = useState<FriendRow[]>([])
  const [loadingFriends, setLoadingFriends] = useState(false)
  const [q, setQ] = useState('')
  const [selected, setSelected] = useState<string[]>([])
  const [name, setName] = useState('')
  const [kind, setKind] = useState<CreateGroupKind>(DEFAULT_GROUP_KIND)
  const [groupId, setGroupId] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const selfId = session?.userId || ''

  useEffect(() => {
    if (!open) return
    setStep('pick')
    setQ('')
    setSelected([])
    setName('')
    setKind(DEFAULT_GROUP_KIND)
    setGroupId('')
    setError('')
    setLoadingFriends(true)
    void fetchFriendList()
      .then((list) => setFriends(list.filter((f) => f.userID && f.userID !== selfId)))
      .catch(() => setFriends([]))
      .finally(() => setLoadingFriends(false))
  }, [open, selfId])

  useEffect(() => {
    if (!open || step !== 'type') return
    if (kind === 'community') {
      setGroupId('')
    } else if (!/^\d{9}$/.test(groupId)) {
      setGroupId(generatePublicGroupId())
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [kind, step, open])

  const filtered = useMemo(() => {
    const key = q.trim().toLowerCase()
    if (!key) return friends
    return friends.filter((f) => {
      const name = `${f.remark || ''} ${f.nick || ''} ${f.userID}`.toLowerCase()
      return name.includes(key)
    })
  }, [friends, q])

  const selectedFriends = useMemo(
    () => friends.filter((f) => selected.includes(f.userID)),
    [friends, selected],
  )

  const allVisibleSelected =
    selected.length >= 100 || (filtered.length > 0 && filtered.every((f) => selected.includes(f.userID)))

  if (!open) return null

  const isCommunity = kind === 'community'

  function displayName(f: FriendRow) {
    return f.remark || f.nick || f.userID
  }

  function toggle(id: string) {
    if (!selected.includes(id) && selected.length >= 100) { setError('最多选择 100 位好友'); return }
    setSelected((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]))
  }

  function toggleSelectAll() {
    if (allVisibleSelected) {
      if (selected.length >= 100) { setSelected([]); return }
      const ids = new Set(filtered.map((f) => f.userID))
      setSelected((prev) => prev.filter((id) => !ids.has(id)))
      return
    }
    setSelected((prev) => {
      const next = new Set(prev)
      filtered.forEach((f) => { if (next.size < 100) next.add(f.userID) })
      return Array.from(next)
    })
  }

  function onRefreshId() {
    if (isCommunity) setGroupId(generateCommunityGroupId())
    else setGroupId(generatePublicGroupId())
  }

  function onGroupIdChange(value: string) {
    if (isCommunity) {
      setGroupId(value.replace(/\s/g, '').slice(0, 48))
      return
    }
    setGroupId(value.replace(/\D/g, '').slice(0, 9))
  }

  function goTypeStep() {
    setError('')
    if (!selected.length) {
      setError('请先选择至少一位好友')
      return
    }
    if (!name.trim()) {
      const first = selectedFriends[0]
      setName(first ? `${displayName(first)}的群聊` : '群聊')
    }
    setStep('type')
  }

  async function handleCreate() {
    setError('')
    if (!name.trim()) {
      setError('请输入群名称')
      return
    }
    if (isCommunity && groupId.trim() && !/^@TGS#/i.test(groupId.trim())) {
      setError('社群群号必须以 @TGS# 开头，或留空由系统自动分配')
      return
    }
    setLoading(true)
    try {
      const res = await createGroupWithTip({
        name: name.trim(),
        groupId: groupId.trim() || undefined,
        kind,
        memberIDList: selected,
        members: selectedFriends.map((f) => ({
          userID: f.userID,
          nick: displayName(f),
        })),
        inviterName: session?.nickname || selfId,
      })
      if (res.warning) window.alert(res.warning)
      onSuccess?.(res.conversationID)
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : '创建失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className={styles.mask} onClick={onClose}>
      <div
        className={step === 'pick' ? styles.split : styles.modal}
        onClick={(e) => e.stopPropagation()}
        role="dialog"
      >
        {step === 'pick' ? (
          <>
            <div className={styles.left}>
              <div className={styles.search}>
                <input
                  value={q}
                  placeholder="请输入userID"
                  onChange={(e) => setQ(e.target.value)}
                />
              </div>
              <button type="button" className={styles.selectAll} onClick={toggleSelectAll}>
                <i className={allVisibleSelected ? styles.checkOn : styles.check} />
                {allVisibleSelected ? '取消全选' : '全选（最多100人）'}
              </button>
              <div className={styles.list}>
                {loadingFriends ? <div className={styles.empty}>加载中…</div> : null}
                {!loadingFriends && !filtered.length ? (
                  <div className={styles.empty}>暂无好友</div>
                ) : null}
                {filtered.map((f) => {
                  const on = selected.includes(f.userID)
                  return (
                    <button
                      key={f.userID}
                      type="button"
                      className={styles.row}
                      onClick={() => toggle(f.userID)}
                    >
                      <i className={on ? styles.checkOn : styles.check} />
                      <div className={styles.avatar}>
                        {f.avatar ? <img src={f.avatar} alt="" /> : displayName(f).slice(0, 1)}
                      </div>
                      <span>{displayName(f)}</span>
                    </button>
                  )
                })}
              </div>
            </div>
            <div className={styles.right}>
              <h3>发起群聊</h3>
              <p className={styles.count}>已选中 {selected.length}/100 人</p>
              <div className={styles.picked}>
                {selectedFriends.map((f) => (
                  <div key={f.userID} className={styles.pickedRow}>
                    <div className={styles.avatar}>
                      {f.avatar ? <img src={f.avatar} alt="" /> : displayName(f).slice(0, 1)}
                    </div>
                    <span className={styles.pickedName}>{displayName(f)}</span>
                    <button type="button" onClick={() => toggle(f.userID)}>
                      ×
                    </button>
                  </div>
                ))}
              </div>
              {error ? <p className={styles.err}>{error}</p> : null}
              <div className={styles.actions}>
                <button type="button" className={styles.cancel} onClick={onClose}>
                  取消
                </button>
                <button type="button" className={styles.done} onClick={goTypeStep}>
                  完成
                </button>
              </div>
            </div>
          </>
        ) : (
          <>
            <header className={styles.head}>
              <button type="button" className={styles.back} onClick={() => setStep('pick')}>
                ‹
              </button>
              <h3>选择群类型</h3>
              <button type="button" onClick={onClose}>
                ✕
              </button>
            </header>
            <div className={styles.body}>
              <p className={styles.pickedHint}>已选 {selected.length} 位好友，创建后将自动拉入群聊</p>
              <label className={styles.field}>
                <span>群名称</span>
                <input
                  value={name}
                  maxLength={30}
                  placeholder="请输入群名称"
                  onChange={(e) => setName(e.target.value)}
                />
              </label>
              <label className={styles.field}>
                <span>群号</span>
                <div className={styles.idRow}>
                  <input
                    value={groupId}
                    maxLength={isCommunity ? 48 : 9}
                    placeholder={
                      isCommunity ? '留空自动分配，或 @TGS# 开头自定义' : '9位数字群号'
                    }
                    onChange={(e) => onGroupIdChange(e.target.value)}
                  />
                  <button type="button" onClick={onRefreshId}>
                    {isCommunity ? '示例' : '刷新'}
                  </button>
                </div>
                <em>
                  {isCommunity
                    ? '社群群号只能是 @TGS# 开头（腾讯 Community）；留空由系统自动生成'
                    : '普通群默认随机 9 位数字，可自定义'}
                </em>
              </label>
              <div className={styles.field}>
                <span>群类型</span>
                <div className={styles.kinds}>
                  {(Object.keys(GROUP_KIND_LABELS) as CreateGroupKind[]).map((k) => (
                    <label key={k} className={`${styles.kind} ${kind === k ? styles.active : ''}`}>
                      <input
                        type="radio"
                        name="group-kind"
                        checked={kind === k}
                        onChange={() => setKind(k)}
                      />
                      <div>
                        <strong>{k === 'community' ? '社群' : '普通群'}</strong>
                        <p>{GROUP_KIND_LABELS[k]}</p>
                      </div>
                    </label>
                  ))}
                </div>
              </div>
              {error ? <p className={styles.err}>{error}</p> : null}
              <button
                type="button"
                className={styles.submit}
                disabled={loading}
                onClick={() => void handleCreate()}
              >
                {loading ? '创建中…' : '创建群聊'}
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  )
}
