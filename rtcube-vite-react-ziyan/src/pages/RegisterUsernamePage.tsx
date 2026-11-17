import { useEffect, useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import {
  getSecurityQuestions,
  getSystemConfig,
  registerByUsername,
  toLocalSession,
  type SecurityQuestion,
} from '@/api/auth'
import { rememberAccount } from '@/api/accounts'
import { acceptPrivacy, isPrivacyAccepted, saveLocalSession } from '@/api/session'
import { PrivacyDialog } from '@/components/PrivacyDialog'
import { APP_NAME } from '@/config/runtime'
import { useAppStore } from '@/store/AppStore'
import styles from './LoginPage.module.css'

export function RegisterUsernamePage() {
  const nav = useNavigate()
  const { bootFromSession } = useAppStore()
  const [username, setUsername] = useState('')
  const [nickname, setNickname] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [inviteCode, setInviteCode] = useState('')
  const [requireInviteCode, setRequireInviteCode] = useState(false)
  const [enableSecurity, setEnableSecurity] = useState(false)
  const [questions, setQuestions] = useState<SecurityQuestion[]>([])
  const [questionId, setQuestionId] = useState(0)
  const [securityAnswer, setSecurityAnswer] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [showPrivacy, setShowPrivacy] = useState(!isPrivacyAccepted())

  useEffect(() => {
    void (async () => {
      const config = await getSystemConfig()
      setRequireInviteCode(Boolean(config.require_invite_code))
      setEnableSecurity(Boolean(config.enable_security_question))
      if (config.enable_security_question) {
        const list = await getSecurityQuestions()
        setQuestions(list)
        if (list[0]) setQuestionId(list[0].id)
      }
    })()
  }, [])

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    if (!isPrivacyAccepted()) {
      setShowPrivacy(true)
      return
    }
    const u = username.trim()
    if (u.length < 4 || u.length > 20) {
      setError('用户名需要 4-20 位')
      return
    }
    if (!/^[a-zA-Z0-9_]+$/.test(u)) {
      setError('用户名仅支持字母、数字、下划线')
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
    if (enableSecurity && (!questionId || !securityAnswer.trim())) {
      setError('请选择密保问题并填写答案')
      return
    }

    setLoading(true)
    try {
      const info = await registerByUsername({
        username: u,
        password,
        nickname: nickname.trim() || u,
        invite_code: inviteCode.trim() || undefined,
        security_question_id: enableSecurity ? questionId : undefined,
        security_answer: enableSecurity ? securityAnswer.trim() : undefined,
      })
      const session = { ...(await toLocalSession(info)), loginAccount: u }
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
          <h1>用户名注册</h1>
          <p>账号密码 · 对接 /v2/auth/register</p>
        </div>
        <form className={styles.form} onSubmit={onSubmit}>
          <input
            className={styles.field}
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="用户名（4-20 位）"
            autoComplete="username"
            maxLength={20}
          />
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
          {enableSecurity ? (
            <>
              <select
                className={styles.field}
                value={questionId}
                onChange={(e) => setQuestionId(Number(e.target.value))}
              >
                {questions.map((q) => (
                  <option key={q.id} value={q.id}>
                    {q.question}
                  </option>
                ))}
              </select>
              <input
                className={styles.field}
                value={securityAnswer}
                onChange={(e) => setSecurityAnswer(e.target.value)}
                placeholder="密保答案"
              />
            </>
          ) : null}
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
