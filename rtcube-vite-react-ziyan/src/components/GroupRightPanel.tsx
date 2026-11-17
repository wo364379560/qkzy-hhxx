import { useCallback, useEffect, useRef, useState } from 'react'
import { GROUP_MEMBER_FIRST_PAGE, GROUP_MEMBER_PAGE_SIZE } from '@/config/runtime'
import { fetchGroupDetail } from '@/im/groupActions'
import { applyMemberLivePatch, onGroupLiveChange } from '@/im/groupLive'
import { fetchGroupMemberPage, fetchGroupMembersUntil, searchGroupMembersPage } from '@/im/ImClient'
import type { GroupMember } from '@/im/types'
import { useAppStore } from '@/store/AppStore'
import {
  GroupMemberContextMenu,
  type MemberMenuState,
} from './GroupMemberContextMenu'
import { GroupNoticeModal } from './GroupNoticeModal'
import { VirtualMemberList } from './VirtualMemberList'
import styles from './GroupRightPanel.module.css'
import iconChevron from '@/assets/icon/chevron_right.svg'

type Props = {
  groupId: string
  onOpenSettings?: () => void
  onOpenMember?: (m: GroupMember, restricted: boolean) => void
}

/**
 * 右栏：公告摘要 + 成员分页虚拟列表。
 * 腾讯 getGroupMemberList 分页：offset/count，回包带下一页 offset。
 */
export function GroupRightPanel({ groupId, onOpenSettings, onOpenMember }: Props) {
  const { session } = useAppStore()
  const [notification, setNotification] = useState('')
  const [name, setName] = useState('')
  const [memberCount, setMemberCount] = useState(0)
  const [ownerID, setOwnerID] = useState('')
  const [blockMemberProfile, setBlockMemberProfile] = useState(false)
  const [selfRole, setSelfRole] = useState<'Owner' | 'Admin' | 'Member'>('Member')
  const [members, setMembers] = useState<GroupMember[]>([])
  const [offset, setOffset] = useState(0)
  const [hasMore, setHasMore] = useState(true)
  const [loading, setLoading] = useState(false)
  const [search, setSearch] = useState('')
  const [searchMode, setSearchMode] = useState(false)
  const [noticeOpen, setNoticeOpen] = useState(false)
  const [menu, setMenu] = useState<MemberMenuState>(null)

  const selfId = session?.userId || ''
  const searchRef = useRef(search)
  searchRef.current = search

  const resetAndLoad = useCallback(
    async (keyword: string) => {
      if (!groupId) return
      setLoading(true)
      try {
        const page = keyword.trim()
          ? await searchGroupMembersPage(groupId, keyword, 0, GROUP_MEMBER_PAGE_SIZE)
          : await fetchGroupMembersUntil(groupId, GROUP_MEMBER_FIRST_PAGE)
        setMembers(page.members)
        setOffset(page.nextOffset)
        setHasMore(page.hasMore)
        setSearchMode(Boolean(keyword.trim()))
        const me = page.members.find((m) => m.userID === selfId)
        if (me) setSelfRole(me.role)
      } catch (e) {
        console.error('group members', e)
        setMembers([])
        setHasMore(false)
      } finally {
        setLoading(false)
      }
    },
    [groupId, selfId],
  )

  useEffect(() => {
    setSearch('')
    setMembers([])
    setOffset(0)
    setHasMore(true)
    setNoticeOpen(false)
    if (!groupId) return
    let cancelled = false
    ;(async () => {
      try {
        const g = await fetchGroupDetail(groupId)
        if (cancelled) return
        setName(g.name || groupId)
        setNotification(g.notification || '')
        setMemberCount(Number(g.memberCount || 0))
        setOwnerID(g.ownerID || '')
        setBlockMemberProfile(g.blockMemberProfile)
        if (selfId && g.ownerID === selfId) setSelfRole('Owner')
      } catch {
        if (!cancelled) {
          setName(groupId)
          setNotification('')
        }
      }
    })()
    return () => {
      cancelled = true
    }
  }, [groupId, selfId])

  useEffect(() => {
    if (!groupId) return
    const applyDetail = () =>
      fetchGroupDetail(groupId)
        .then((g) => {
          setBlockMemberProfile(g.blockMemberProfile)
          setName(g.name || groupId)
          setNotification(g.notification || '')
          setMemberCount(Number(g.memberCount || 0))
          setOwnerID(g.ownerID || '')
        })
        .catch(() => undefined)
    const off = onGroupLiveChange((e) => {
      if (e.groupID !== groupId) return
      if (e.blockMemberProfile !== undefined) {
        setBlockMemberProfile(e.blockMemberProfile)
      }
      if (e.added?.length || e.removedIds?.length || e.updated?.length) {
        setMembers((prev) => {
          const { next, addedCount, removedCount } = applyMemberLivePatch(prev, e)
          if (addedCount || removedCount) {
            setMemberCount((n) => Math.max(0, n + addedCount - removedCount))
          }
          return next
        })
        return
      }
      if (!e.membersChanged) void applyDetail()
    })
    const poll = window.setInterval(() => {
      void applyDetail()
    }, 2000)
    return () => {
      off()
      window.clearInterval(poll)
    }
  }, [groupId, resetAndLoad])

  useEffect(() => {
    if (!groupId) return
    const t = window.setTimeout(() => {
      void resetAndLoad(search)
    }, 280)
    return () => window.clearTimeout(t)
  }, [groupId, search, resetAndLoad])

  const loadMore = useCallback(async () => {
    if (!groupId || loading || !hasMore) return
    setLoading(true)
    try {
      const page = searchMode
        ? await searchGroupMembersPage(groupId, search, offset, GROUP_MEMBER_PAGE_SIZE)
        : await fetchGroupMemberPage(groupId, offset, GROUP_MEMBER_PAGE_SIZE)
      setMembers((prev) => {
        const seen = new Set(prev.map((m) => m.userID))
        const merged = [...prev]
        for (const m of page.members) {
          if (!seen.has(m.userID)) merged.push(m)
        }
        return merged
      })
      setOffset(page.nextOffset)
      setHasMore(page.hasMore)
    } catch (e) {
      console.error(e)
      setHasMore(false)
    } finally {
      setLoading(false)
    }
  }, [groupId, loading, hasMore, searchMode, search, offset])

  function memberRestricted(m: GroupMember) {
    const isManager = selfRole === 'Owner' || selfRole === 'Admin'
    const targetManager = m.role === 'Owner' || m.role === 'Admin' || m.userID === ownerID
    return blockMemberProfile && !isManager && !targetManager && m.userID !== selfId
  }

  function handlePick(m: GroupMember) {
    onOpenMember?.(m, memberRestricted(m))
  }

  const canEditNotice = selfRole === 'Owner' || selfRole === 'Admin'

  return (
    <aside className={styles.pane}>
      <header className={styles.head}>
        <div>
          <strong>{name || '群资料'}</strong>
          <span className={styles.sub}>{memberCount ? `${memberCount} 人` : '群成员'}</span>
        </div>
        <button type="button" className={styles.link} onClick={onOpenSettings}>
          详情
        </button>
      </header>

      <button
        type="button"
        className={styles.noticeCard}
        onClick={() => setNoticeOpen(true)}
      >
        <div className={styles.noticeHead}>
          <span>群公告</span>
          <img src={iconChevron} alt="" />
        </div>
        <p className={styles.noticeBody}>
          {notification?.trim() ? notification : '暂无公告'}
        </p>
      </button>

      <GroupNoticeModal
        open={noticeOpen}
        groupId={groupId}
        canEdit={canEditNotice}
        initial={notification}
        onClose={() => setNoticeOpen(false)}
        onSaved={(content) => setNotification(content)}
      />

      <div className={styles.searchWrap}>
        <input
          className={styles.search}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="搜索成员（分页，不拉全员）"
        />
      </div>

      <VirtualMemberList
        members={members}
        hasMore={hasMore}
        loading={loading}
        onLoadMore={() => void loadMore()}
        onPick={handlePick}
        onContextMenu={(e, m) => setMenu({ x: e.clientX, y: e.clientY, member: m })}
        emptyText={search.trim() ? '无匹配成员' : '暂无成员'}
      />
      <GroupMemberContextMenu
        menu={menu}
        groupId={groupId}
        selfId={selfId}
        selfRole={selfRole}
        restricted={menu ? memberRestricted(menu.member) : false}
        onClose={() => setMenu(null)}
        onChanged={() => undefined}
      />
    </aside>
  )
}
