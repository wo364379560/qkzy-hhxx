import { useNavigate } from 'react-router-dom'
import { useI18n } from '@/i18n/I18nProvider'
import { useAppStore } from '@/store/AppStore'
import { AccountSwitcher } from './AccountSwitcher'
import styles from './SideNav.module.css'

export type MainTab =
  | 'messages'
  | 'contacts'
  | 'discover'
  | 'files'
  | 'favorites'
  | 'shortcuts'
  | 'mine'

type Props = {
  active: MainTab
  onChange: (tab: MainTab) => void
  onAvatarClick?: () => void
}

const TABS: Array<{
  key: MainTab
  label:
    | 'tab_messages'
    | 'tab_contacts'
    | 'tab_discover'
    | 'tab_files'
    | 'tab_favorites'
    | 'tab_shortcuts'
    | 'tab_mine'
  icon: string
}> = [
  { key: 'messages', label: 'tab_messages', icon: '💬' },
  { key: 'contacts', label: 'tab_contacts', icon: '👥' },
  { key: 'files', label: 'tab_files', icon: '📁' },
  { key: 'favorites', label: 'tab_favorites', icon: '⭐' },
  { key: 'shortcuts', label: 'tab_shortcuts', icon: '/' },
  { key: 'mine', label: 'tab_mine', icon: '👤' },
]

/** QQ 式最左侧竖栏 */
export function SideNav({ active, onChange, onAvatarClick }: Props) {
  const { t, lang, setLang } = useI18n()
  const { session, ready, connecting, connectError, reconnect, logout, totalUnread, contactBadge } =
    useAppStore()
  const nav = useNavigate()
  const name = session?.nickname || session?.userId || '?'
  const avatar = session?.avatar || ''

  function tabBadge(key: MainTab) {
    if (key === 'messages' && totalUnread > 0) return totalUnread > 99 ? '99+' : String(totalUnread)
    if (key === 'contacts' && contactBadge > 0)
      return contactBadge > 99 ? '99+' : String(contactBadge)
    return ''
  }

  async function onLogout() {
    if (!confirm(t('logout_confirm'))) return
    await logout()
    nav('/login', { replace: true })
  }

  return (
    <nav className={styles.nav}>
      <button
        type="button"
        className={styles.avatarBtn}
        onClick={() => {
          onChange('mine')
          onAvatarClick?.()
        }}
        title={name}
      >
        <div className={styles.avatar}>
          {avatar ? <img src={avatar} alt="" /> : name.slice(0, 1)}
        </div>
      </button>

      <div className={styles.tabs}>
        {TABS.map((tab) => (
          <button
            key={tab.key}
            type="button"
            className={`${styles.tab} ${active === tab.key ? styles.active : ''}`}
            onClick={() => onChange(tab.key)}
            title={t(tab.label)}
          >
            <span className={styles.iconWrap}>
              <span className={styles.icon}>{tab.icon}</span>
              {tabBadge(tab.key) ? <i className={styles.tabBadge}>{tabBadge(tab.key)}</i> : null}
            </span>
            <span className={styles.label}>{t(tab.label)}</span>
          </button>
        ))}
      </div>

      <AccountSwitcher />

      <div className={styles.foot}>
        <div
          className={`${styles.status} ${ready ? styles.ok : connectError ? styles.err : styles.wait}`}
          title={connectError || (ready ? t('connected') : t('connecting'))}
        >
          {ready ? '●' : connecting ? '…' : connectError ? '!' : '○'}
        </div>
        {!ready && connectError ? (
          <button type="button" className={styles.mini} onClick={() => void reconnect()}>
            {t('reconnect')}
          </button>
        ) : null}
        <button
          type="button"
          className={styles.mini}
          onClick={() => setLang(lang === 'zh-CN' ? 'en-US' : 'zh-CN')}
          title={t('language')}
        >
          {lang === 'zh-CN' ? 'EN' : '中'}
        </button>
        <button type="button" className={styles.logoutBtn} onClick={() => void onLogout()} title={t('logout')}>
          {t('logout_short')}
        </button>
      </div>
    </nav>
  )
}
