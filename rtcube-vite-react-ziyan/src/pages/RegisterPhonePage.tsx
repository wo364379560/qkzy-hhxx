import { useEffect, useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import {
  getSystemConfig,
  registerByPhone,
  sendSmsCode,
  toLocalSession,
} from '@/api/auth'
import { rememberAccount } from '@/api/accounts'
import { acceptPrivacy, isPrivacyAccepted, saveLocalSession } from '@/api/session'
import { PrivacyDialog } from '@/components/PrivacyDialog'
import { APP_NAME } from '@/config/runtime'
import { useAppStore } from '@/store/AppStore'
import styles from './LoginPage.module.css'

function isValidPhone(phone: string) {
  return /^1[3-9]\d{9}$/.test(phone)
}

export function RegisterPhonePage() {
  const nav = useNavigate()
  const { bootFromSession } = useAppStore()
  const [phone, setPhone] = useState('')
  const [smsCode, setSmsCode] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [nickname, setNickname] = useState('')
  const [inviteCode, setInviteCode] = useState('')
  const [requireInviteCode, setRequireInviteCode] = useState(false)
  const [countdown, setCountdown] = useState(0)
  const [sending, setSending] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [showPrivacy, setShowPrivacy] = useState(!isPrivacyAccepted())

  useEffect(() => {
    void getSystemConfig().then((c) => setRequireInviteCode(Boolean(c.require_invite_code)))
  }, [])

  useEffect(() => {
    if (countdown <= 0) return
    const t = window.setTimeout(() => setCountdown((c) => c - 1), 1000)
    return () => clearTimeout(t)
  }, [countdown])

  async function onSendSms() {
    setError('')
    if (!isValidPhone(phone)) {
      setError('请输入正确的手机号')
      return
    }
    setSending(true)
    try {
      await sendSmsCode(phone.trim(), 'register')
      setCountdown(60)
    } catch (e) {
      setError(e instanceof Error ? e.message : '验证码发送失败')
    } finally {
      setSending(false)
    }
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    if (!isPrivacyAccepted()) {
      setShowPrivacy(true)
      return
    }
    if (!isValidPhone(phone)) {
      setError('请输入正确的手机号')
      return
    }
    if (!smsCode.trim() || smsCode.trim().length < 4) {
      setError('请输入短信验证码')
      return
    }
    if (password.length < 6) {
      setError('密码至少 6 位')
      return
    }
    if (password !== confirmPassword) {
      setError('两次密码不一致')
      return
    }
    if (requireInviteCode && !inviteCode.trim()) {
      setError('请输入邀请码')
      return
    }

    setLoading(true)
    try {
      const info = await registerByPhone({
        phone: phone.trim(),
        sms_code: smsCode.trim(),
        password,
        nickname: nickname.trim() || undefined,
        invite_code: inviteCode.trim() || undefined,
      })
      if (!info.user_sig || !info.sdk_app_id) throw new Error('注册成功但缺少 IM 凭证')
      const session = { ...(await toLocalSession(info)), loginAccount: phone.trim() }
      saveLocalSession(session)
      await rememberAccount(session).catch((vaultError) => {
        console.warn('remember account failed', vaultError)
      })
      await bootFromSession(session)
      nav('/chat', { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : '注册失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className={styles.page}>
      <div className={styles.card}>
        <div className={styles.brand}>
          <div className={styles.logo}>{APP_NAME.slice(0, 1)}</div>
          <h1>手机号注册</h1>
          <p>短信验证码 · 对接 /v2/auth/register</p>
        </div>
        <form className={styles.form} onSubmit={onSubmit}>
          <input
            className={styles.field}
            type="tel"
            value={phone}
            onChange={(e) => setPhone(e.target.value)}
            placeholder="手机号"
            maxLength={11}
            autoComplete="tel"
          />
          <div className={styles.smsRow}>
            <input
              className={styles.field}
              value={smsCode}
              onChange={(e) => setSmsCode(e.target.value.replace(/\D/g, ''))}
              placeholder="验证码"
              maxLength={6}
            />
            <button
              type="button"
              className={styles.smsBtn}
              disabled={sending || countdown > 0}
              onClick={() => void onSendSms()}
            >
              {countdown > 0 ? `${countdown}s` : sending ? '发送中' : '获取验证码'}
            </button>
          </div>
          <input
            className={styles.field}
            value={nickname}
            onChange={(e) => setNickname(e.target.value)}
            placeholder="昵称（可选）"
          />
          <input
            className={styles.field}
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="密码（至少 6 位）"
            autoComplete="new-password"
          />
          <input
            className={styles.field}
            type="password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            placeholder="确认密码"
            autoComplete="new-password"
          />
          <input
            className={styles.field}
            value={inviteCode}
            onChange={(e) => setInviteCode(e.target.value)}
            placeholder={requireInviteCode ? '邀请码（必填）' : '邀请码（可选）'}
          />
          {error ? <p className={styles.err}>{error}</p> : null}
          <button className={styles.submit} type="submit" disabled={loading}>
            {loading ? '注册中…' : '注册并登录'}
          </button>
          <p className={styles.footer}>
            <Link to="/register">其他注册方式</Link>
            {' · '}
            <Link to="/login">返回登录</Link>
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
          window.close()
          setError('需同意协议后使用。请关闭后重新打开。')
          setShowPrivacy(true)
        }}
      />
    </div>
  )
}
