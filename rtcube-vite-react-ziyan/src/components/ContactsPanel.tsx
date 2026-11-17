import { useCallback, useEffect, useState } from 'react'
import {
  acceptFriendApplication,
  fetchBlacklistProfiles,
  fetchFriendApplications,
  fetchGroupApplications,
  fetchJoinedGroups,
  handleGroupApplication,
  refuseFriendApplication,
  type FriendApplication,
  type GroupApplication,
  type JoinedGroup,
} from '@/im/friends'
import { fetchFriendList, type FriendRow } from '@/im/ImClient'
import { removeFromBlacklist } from '@/im/profileActions'
import { useAppStore } from '@/store/AppStore'
import styles from './ContactsPanel.module.css'

type Props = {
  onOpenChat: (userId: string) => void
  onOpenGroup: (groupId: string) => void
  onOpenProfile?: (userId: string, nickname?: string, avatar?: string) => void
}

type SubView = 'home' | 'newFriends' | 'groups' | 'blacklist'

export function ContactsPanel({ onOpenChat, onOpenGroup, onOpenProfile }: Props) {
  const { ready, contactBadge, refreshContactBadge } = useAppStore()
  const [view, setView] = useState<SubView>('home')
  const [friends, setFriends] = useState<FriendRow[]>([])
  const [friendApps, setFriendApps] = useState<FriendApplication[]>([])
  const [groupApps, setGroupApps] = useState<GroupApplication[]>([])
  const [groups, setGroups] = useState<JoinedGroup[]>([])
  const [blacklist, setBlacklist] = useState<Array<{ userID: string; nick: string; avatar: string }>>(
    [],
  )
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [newTab, setNewTab] = useState<'person' | 'group'>('person')

  const loadHome = useCallback(async () => {
    if (!ready) {
      setError('连接中…')
      return
    }
    setLoading(true)
    setError('')
    try {
      const [f, fa, ga] = await Promise.all([
        fetchFriendList(),
        fetchFriendApplications().catch(() => []),
        fetchGroupApplications().catch(() => []),
      ])
      setFriends(f)
      setFriendApps(fa)
      setGroupApps(ga)
      void refreshContactBadge()
    } catch (e) {
      setError(e instanceof Error ? e.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }, [ready, refreshContactBadge])

  const loadGroups = useCallback(async () => {
    setLoading(true)
    try {
      setGroups(await fetchJoinedGroups())
    } catch (e) {
      setError(e instanceof Error ? e.message : '群组加载失败')
    } finally {
      setLoading(false)
    }
  }, [])

  const loadBlacklist = useCallback(async () => {
    setLoading(true)
    try {
      setBlacklist(await fetchBlacklistProfiles())
    } catch (e) {
      setError(e instanceof Error ? e.message : '黑名单加载失败')
    } finally {
      setLoading(false)
    }
  }, [])

  const loadNewFriends = useCallback(async () => {
    setLoading(true)
    try {
      const [fa, ga] = await Promise.all([
        fetchFriendApplications().catch(() => []),
        fetchGroupApplications().catch(() => []),
      ])
      setFriendApps(fa)
      setGroupApps(ga)
      void refreshContactBadge()
    } finally {
      setLoading(false)
    }
  }, [refreshContactBadge])

  useEffect(() => {
    if (view === 'home') void loadHome()
    else if (view === 'groups') void loadGroups()
    else if (view === 'blacklist') void loadBlacklist()
    else if (view === 'newFriends') void loadNewFriends()
  }, [view, loadHome, loadGroups, loadBlacklist, loadNewFriends])

  const newCount = friendApps.length + groupApps.length

  if (view === 'newFriends') {
    return (
      <div className={styles.pane}>
        <header className={styles.subHead}>
          <button type="button" className={styles.back} onClick={() => setView('home')}>
            ‹
          </button>
          <strong>新的朋友</strong>
        </header>
        <div className={styles.seg}>
          <button
            type="button"
            className={newTab === 'person' ? styles.segOn : ''}
            onClick={() => setNewTab('person')}
          >
            个人 {friendApps.length ? `(${friendApps.length})` : ''}
          </button>
          <button
            type="button"
            className={newTab === 'group' ? styles.segOn : ''}
            onClick={() => setNewTab('group')}
          >
            群 {groupApps.length ? `(${groupApps.length})` : ''}
          </button>
        </div>
        {loading ? <div className={styles.empty}>加载中…</div> : null}
        {newTab === 'person' ? (
          <ul className={styles.list}>
            {friendApps.map((a) => (
              <li key={a.userID} className={styles.appRow}>
                <button
                  type="button"
                  className={styles.row}
                  onClick={() => onOpenProfile?.(a.userID, a.nick, a.avatar)}
                >
                  <div className={styles.avatar}>
                    {a.avatar ? <img src={a.avatar} alt="" /> : (a.nick || a.userID).slice(0, 1)}
                  </div>
                  <div className={styles.meta}>
                    <strong>{a.nick || a.userID}</strong>
                    <span>{a.wording || '请求添加你为好友'}</span>
                  </div>
                </button>
                <div className={styles.appBtns}>
                  <button
                    type="button"
                    className={styles.ok}
                    onClick={() =>
                      void acceptFriendApplication(a.userID).then(loadNewFriends).catch((e) =>
                        alert(e instanceof Error ? e.message : '同意失败'),
                      )
                    }
                  >
                    同意
                  </button>
                  <button
                    type="button"
                    className={styles.no}
                    onClick={() =>
                      void refuseFriendApplication(a.userID).then(loadNewFriends).catch((e) =>
                        alert(e instanceof Error ? e.message : '拒绝失败'),
                      )
                    }
                  >
                    拒绝
                  </button>
                </div>
              </li>
            ))}
            {!loading && !friendApps.length ? <li className={styles.empty}>暂无个人好友申请</li> : null}
          </ul>
        ) : (
          <ul className={styles.list}>
            {groupApps.map((a) => (
              <li key={`${a.groupID}_${a.applicant}`} className={styles.appRow}>
                <div className={styles.row}>
                  <div className={styles.avatar}>
                    {a.avatar ? <img src={a.avatar} alt="" /> : (a.nick || a.applicant).slice(0, 1)}
                  </div>
                  <div className={styles.meta}>
                    <strong>{a.nick || a.applicant}</strong>
                    <span>
                      申请加入 {a.groupName}
                      {a.wording ? ` · ${a.wording}` : ''}
                    </span>
                  </div>
                </div>
                <div className={styles.appBtns}>
                  <button
                    type="button"
                    className={styles.ok}
                    onClick={() =>
                      void handleGroupApplication(a, 'Agree')
                        .then(loadNewFriends)
                        .catch((e) => alert(e instanceof Error ? e.message : '同意失败'))
                    }
                  >
                    同意
                  </button>
                  <button
                    type="button"
                    className={styles.no}
                    onClick={() =>
                      void handleGroupApplication(a, 'Reject')
                        .then(loadNewFriends)
                        .catch((e) => alert(e instanceof Error ? e.message : '拒绝失败'))
                    }
                  >
                    拒绝
                  </button>
                </div>
              </li>
            ))}
            {!loading && !groupApps.length ? <li className={styles.empty}>暂无入群申请</li> : null}
          </ul>
        )}
      </div>
    )
  }

  if (view === 'groups') {
    return (
      <div className={styles.pane}>
        <header className={styles.subHead}>
          <button type="button" className={styles.back} onClick={() => setView('home')}>
            ‹
          </button>
          <strong>群组</strong>
        </header>
        {loading ? <div className={styles.empty}>加载中…</div> : null}
        <ul className={styles.list}>
          {groups.map((g) => (
            <li key={g.groupID}>
              <button type="button" className={styles.row} onClick={() => onOpenGroup(g.groupID)}>
                <div className={`${styles.avatar} ${styles.groupAv}`}>
                  {g.avatar ? <img src={g.avatar} alt="" /> : g.name.slice(0, 1)}
                </div>
                <div className={styles.meta}>
                  <strong>{g.name}</strong>
                  <span>{g.memberCount} 人 · {g.groupID}</span>
                </div>
              </button>
            </li>
          ))}
          {!loading && !groups.length ? <li className={styles.empty}>暂无群组</li> : null}
        </ul>
      </div>
    )
  }

  if (view === 'blacklist') {
    return (
      <div className={styles.pane}>
        <header className={styles.subHead}>
          <button type="button" className={styles.back} onClick={() => setView('home')}>
            ‹
          </button>
          <strong>黑名单</strong>
        </header>
        {loading ? <div className={styles.empty}>加载中…</div> : null}
        <ul className={styles.list}>
          {blacklist.map((u) => (
            <li key={u.userID} className={styles.appRow}>
              <button
                type="button"
                className={styles.row}
                onClick={() => onOpenProfile?.(u.userID, u.nick, u.avatar)}
              >
                <div className={styles.avatar}>
                  {u.avatar ? <img src={u.avatar} alt="" /> : (u.nick || u.userID).slice(0, 1)}
                </div>
                <div className={styles.meta}>
                  <strong>{u.nick || u.userID}</strong>
                  <em>{u.userID}</em>
                </div>
              </button>
              <button
                type="button"
                className={styles.no}
                onClick={() =>
                  void removeFromBlacklist(u.userID)
                    .then(loadBlacklist)
                    .catch((e) => alert(e instanceof Error ? e.message : '移除失败'))
                }
              >
                移除
              </button>
            </li>
          ))}
          {!loading && !blacklist.length ? <li className={styles.empty}>黑名单为空</li> : null}
        </ul>
      </div>
    )
  }

  const badge = contactBadge || newCount

  return (
    <div className={styles.pane}>
      <header className={styles.head}>
        <div>
          <strong>联系人</strong>
          <span className={styles.sub}>
            {friends.length ? `好友 ${friends.length}` : '通讯录'}
          </span>
        </div>
        <button type="button" className={styles.refresh} onClick={() => void loadHome()} disabled={loading}>
          刷新
        </button>
      </header>

      <div className={styles.entries}>
        <button type="button" className={styles.entry} onClick={() => setView('newFriends')}>
          <span className={`${styles.entryIcon} ${styles.orange}`}>新</span>
          <span className={styles.entryTitle}>新的朋友</span>
          {badge > 0 ? <span className={styles.badge}>{badge > 99 ? '99+' : badge}</span> : null}
          <span className={styles.chev}>›</span>
        </button>
        <button type="button" className={styles.entry} onClick={() => setView('groups')}>
          <span className={`${styles.entryIcon} ${styles.green}`}>群</span>
          <span className={styles.entryTitle}>群组</span>
          <span className={styles.chev}>›</span>
        </button>
        <button type="button" className={styles.entry} onClick={() => setView('blacklist')}>
          <span className={`${styles.entryIcon} ${styles.gray}`}>黑</span>
          <span className={styles.entryTitle}>黑名单</span>
          <span className={styles.chev}>›</span>
        </button>
      </div>

      <div className={styles.section}>好友 {friends.length ? `(${friends.length})` : ''}</div>
      {loading ? <div className={styles.empty}>加载中…</div> : null}
      {!loading && error ? <div className={styles.err}>{error}</div> : null}

      <ul className={styles.list}>
        {friends.map((f) => {
          const title = f.remark || f.nick || f.userID
          return (
            <li key={f.userID}>
              <button
                type="button"
                className={styles.row}
                onClick={() => onOpenProfile?.(f.userID, f.nick, f.avatar)}
              >
                <div className={styles.avatar}>
                  {f.avatar ? <img src={f.avatar} alt="" /> : title.slice(0, 1)}
                </div>
                <div className={styles.meta}>
                  <strong>{title}</strong>
                  {f.remark && f.nick ? <span>昵称 {f.nick}</span> : null}
                </div>
              </button>
            </li>
          )
        })}
        {!loading && !error && !friends.length ? (
          <li className={styles.empty}>暂无好友</li>
        ) : null}
      </ul>
    </div>
  )
}
