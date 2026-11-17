import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { GroupDetailDrawer } from '@/components/GroupDetailDrawer'
import { UserProfileDrawer } from '@/components/UserProfileDrawer'

type UserTarget = {
  userId: string
  nickname?: string
  avatar?: string
  restricted?: boolean
}

type DetailUiValue = {
  openUserProfile: (t: UserTarget) => void
  openGroupDetail: (groupId: string) => void
  navigateToChat: (conversationID: string) => void
}

const DetailUiContext = createContext<DetailUiValue | null>(null)

export function DetailUiProvider({
  children,
  onNavigateToChat,
}: {
  children: ReactNode
  onNavigateToChat?: (conversationID: string) => void
}) {
  const [user, setUser] = useState<UserTarget | null>(null)
  const [groupId, setGroupId] = useState('')

  const openUserProfile = useCallback((t: UserTarget) => {
    setGroupId('')
    setUser(t)
  }, [])

  const openGroupDetail = useCallback((id: string) => {
    setUser(null)
    setGroupId(id)
  }, [])

  const navigateToChat = useCallback(
    (conversationID: string) => {
      setUser(null)
      setGroupId('')
      onNavigateToChat?.(conversationID)
    },
    [onNavigateToChat],
  )

  const value = useMemo(
    () => ({ openUserProfile, openGroupDetail, navigateToChat }),
    [openUserProfile, openGroupDetail, navigateToChat],
  )

  return (
    <DetailUiContext.Provider value={value}>
      <div style={{ position: 'relative', height: '100%' }}>
        {children}
        <UserProfileDrawer
          open={Boolean(user)}
          userId={user?.userId || ''}
          nickname={user?.nickname}
          avatar={user?.avatar}
          restricted={user?.restricted}
          onClose={() => setUser(null)}
        />
        <GroupDetailDrawer
          open={Boolean(groupId)}
          groupId={groupId}
          onClose={() => setGroupId('')}
        />
      </div>
    </DetailUiContext.Provider>
  )
}

export function useDetailUi() {
  const ctx = useContext(DetailUiContext)
  if (!ctx) throw new Error('useDetailUi must be used within DetailUiProvider')
  return ctx
}
