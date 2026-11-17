import { useState, type FormEvent } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { rememberAccount } from '@/api/accounts'
import { loginByPassword, toLocalSession } from '@/api/auth'
import {
  acceptPrivacy,
  isPrivacyAccepted,
  saveLocalSession,
} from '@/api/session'
import { PrivacyDialog } from '@/components/PrivacyDialog'
import { APP_NAME } from '@/config/runtime'
import { useAppStore } from '@/store/AppStore'
import styles from './LoginPage.module.css'

export function LoginPage() {
  const nav = useNavigate()
  const [searchParams] = useSearchParams()
  const addingAccount = searchParams.get('addAccount') === '1'
  const { bootFromSession } = useAppStore()
  const [username, setUsername] = useState(() => searchParams.get('account') || '')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [showPrivacy, setShowPrivacy] = useState(!isPrivacyAccepted())

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    if (!isPrivacyAccepted()) {
      setShowPrivacy(true)
      return
    }
    if (!username.trim() || !password) {
      setError('请输入手机号和密码')
      return
    }
    setLoading(true)
    try {
      const info = await loginByPassword(username.trim(), password)
      if (!info.user_sig) throw new Error('登录成功但未返回 user_sig')
      if (!info.sdk_app_id) throw new Error('登录成功但未返回 sdk_app_id')
      const session = { ...(await toLocalSession(info)), loginAccount: username.trim() }
      saveLocalSession(session)
      await rememberAccount(session).catch((vaultError) => {
        console.warn('remember account failed', vaultError)
      })
      await bootFromSession(session)
      nav('/chat', { replace: true })
    } catch (err) {
      console.error('login failed', err)
      setError(err instanceof Error ? err.message : '登录失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className={styles.page}>
      <div className={styles.card}>
        <div className={styles.brand}>
          <div className={styles.logo}>{APP_NAME.slice(0, 1)}</div>
          <h1>{addingAccount ? '添加账号' : APP_NAME}</h1>
          {addingAccount ? <p>登录成功后将自动切换到新账号</p> : null}
        </div>
        <form className={styles.form} onSubmit={onSubmit}>
          <input
            className={styles.field}
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="手机号"
            inputMode="tel"
            autoComplete="tel"
          />
          <input
            className={styles.field}
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="密码"
            autoComplete="current-password"
          />
          {error ? <p className={styles.err}>{error}</p> : null}
          <button className={styles.submit} type="submit" disabled={loading}>
            {loading ? '登录中…' : '登录'}
          </button>
          <p className={styles.footer}>
            {addingAccount ? (
              <button type="button" className={styles.linkButton} onClick={() => nav('/chat', { replace: true })}>
                取消，返回当前账号
              </button>
            ) : (
              <>
                没有账号？<Link to="/register">手机号 / 用户名注册</Link>
              </>
            )}
          </p>
        </form>
      </div>

      <PrivacyDialog
        open={showPrivacy}
        onAgree={() => {
          acceptPrivacy()
          setShowPrivacy(false)
        }}
        onDisagree={() => {
          // 规格：暂不使用 = 关；不写已同意
          window.close()
          setError('需同意协议后使用。请关闭后重新打开。')
          setShowPrivacy(true)
        }}
      />
    </div>
  )
}
