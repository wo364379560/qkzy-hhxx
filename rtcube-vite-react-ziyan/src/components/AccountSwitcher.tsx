import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  listSavedAccounts,
  restoreSavedAccount,
  type SavedAccount,
} from '@/api/accounts'
import { useI18n } from '@/i18n/I18nProvider'
import { useAppStore } from '@/store/AppStore'
import styles from './AccountSwitcher.module.css'

export function AccountSwitcher() {
  const nav = useNavigate()
  const { t } = useI18n()
  const { session, connecting, switchAccount } = useAppStore()
  const rootRef = useRef<HTMLDivElement>(null)
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const [switchingId, setSwitchingId] = useState('')
  const [failedAccount, setFailedAccount] = useState<SavedAccount | null>(null)
  const [error, setError] = useState('')
  const [accounts, setAccounts] = useState<SavedAccount[]>([])

  useEffect(() => {
    if (!open) return
    const close = (event: PointerEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) setOpen(false)
    }
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setOpen(false)
    }
    document.addEventListener('pointerdown', close)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('pointerdown', close)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  async function loadAccounts() {
    setLoading(true)
    setError('')
    try {
      setAccounts(await listSavedAccounts())
    } catch (err) {
      setError(err instanceof Error ? err.message : t('accounts_load_fail'))
    } finally {
      setLoading(false)
    }
  }

  function openLogin(account?: SavedAccount) {
    const query = new URLSearchParams({ addAccount: '1' })
    if (account?.loginAccount) query.set('account', account.loginAccount)
    setOpen(false)
    nav(`/login?${query.toString()}`)
  }

  async function chooseAccount(account: SavedAccount) {
    if (account.userId === session?.userId) {
      setOpen(false)
      return
    }
    if (!account.refreshToken) {
      openLogin(account)
      return
    }
    setSwitchingId(account.userId)
    setFailedAccount(null)
    setError('')
    try {
      const restored = await restoreSavedAccount(account)
      await switchAccount(restored)
      setOpen(false)
    } catch (err) {
      setFailedAccount(account)
      setError(err instanceof Error ? err.message : t('account_switch_fail'))
    } finally {
      setSwitchingId('')
    }
  }

  return (
    <div className={styles.slot} ref={rootRef}>
      <button
        type="button"
        className={`${styles.trigger} ${open ? styles.triggerActive : ''}`}
        onClick={() => {
          const next = !open
          setOpen(next)
          if (next) void loadAccounts()
        }}
        title={t('account_manager')}
        aria-expanded={open}
      >
        <span className={styles.triggerIcon}>⇄</span>
        <span>{t('account_short')}</span>
      </button>

      {open ? (
        <section className={styles.panel} aria-label={t('account_manager')}>
          <header className={styles.header}>
            <div>
              <strong>{t('account_manager')}</strong>
              <p>{t('account_manager_hint')}</p>
            </div>
            <button type="button" className={styles.close} onClick={() => setOpen(false)} aria-label="关闭">
              ×
            </button>
          </header>

          <div className={styles.list}>
            {loading ? <div className={styles.empty}>{t('loading')}</div> : null}
            {!loading && !accounts.length ? (
              <div className={styles.empty}>{t('accounts_empty')}</div>
            ) : null}
            {!loading
              ? accounts.map((account) => {
                  const current = account.userId === session?.userId
                  const switching = switchingId === account.userId
                  const name = account.nickname || account.userId
                  return (
                    <button
                      type="button"
                      className={`${styles.account} ${current ? styles.current : ''}`}
                      key={account.userId}
                      disabled={Boolean(switchingId) || connecting}
                      onClick={() => void chooseAccount(account)}
                    >
                      <span className={styles.avatar}>
                        {account.avatar ? <img src={account.avatar} alt="" /> : name.slice(0, 1)}
                      </span>
                      <span className={styles.identity}>
                        <strong>{name}</strong>
                        <small>{account.userId}</small>
                      </span>
                      <span className={styles.state}>
                        {switching ? t('account_switching') : current ? '✓' : ''}
                      </span>
                    </button>
                  )
                })
              : null}
          </div>

          {error ? (
            <div className={styles.error}>
              <span>{error}</span>
              {failedAccount ? (
                <button type="button" onClick={() => openLogin(failedAccount)}>
                  {t('account_relogin')}
                </button>
              ) : null}
            </div>
          ) : null}

          <button type="button" className={styles.add} onClick={() => openLogin()}>
            <span>＋</span>
            {t('account_add')}
          </button>
          <p className={styles.security}>{t('account_security_hint')}</p>
        </section>
      ) : null}
    </div>
  )
}
