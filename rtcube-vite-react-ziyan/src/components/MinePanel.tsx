import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { deleteAccount, fetchProfile, updateProfile } from '@/api/auth'
import { uploadAvatarFile } from '@/api/media'
import { saveLocalSession } from '@/api/session'
import { updateMyImProfile } from '@/im/ImClient'
import { useI18n } from '@/i18n/I18nProvider'
import { useAppStore } from '@/store/AppStore'
import styles from './MinePanel.module.css'

type EditKey = 'nick' | 'sign' | 'gender' | 'birthday' | 'wanan' | null

function genderLabel(g: number) {
  if (g === 1) return '男'
  if (g === 2) return '女'
  return '未知'
}

function birthdayLabel(n?: number) {
  const s = String(n || '')
  if (s.length !== 8) return '未知'
  return `${s.slice(0, 4)}-${s.slice(4, 6)}-${s.slice(6, 8)}`
}

function birthdayToInput(n?: number) {
  const s = String(n || '')
  if (s.length !== 8) return ''
  return `${s.slice(0, 4)}-${s.slice(4, 6)}-${s.slice(6, 8)}`
}

export function MinePanel() {
  const { t } = useI18n()
  const { session, setSession, logout } = useAppStore()
  const nav = useNavigate()
  const fileRef = useRef<HTMLInputElement>(null)
  const [busy, setBusy] = useState(false)
  const [tip, setTip] = useState('')
  const [loading, setLoading] = useState(true)
  const [nickname, setNickname] = useState(session?.nickname || '')
  const [avatar, setAvatar] = useState(session?.avatar || '')
  const [signature, setSignature] = useState('')
  const [gender, setGender] = useState(0)
  const [birthday, setBirthday] = useState(0)
  const [wananId, setWananId] = useState('')
  const [userId, setUserId] = useState(session?.userId || '')
  const [edit, setEdit] = useState<EditKey>(null)
  const [draft, setDraft] = useState('')

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        const p = await fetchProfile()
        if (cancelled) return
        setNickname(p.nickname || session?.nickname || '')
        setAvatar(p.avatar_url || session?.avatar || '')
        setSignature(p.self_signature || '')
        setGender(Number(p.gender || 0))
        setBirthday(Number(p.birthday || 0))
        setWananId(p.wanan_id || '')
        setUserId(p.im_user_id || session?.userId || '')
      } catch (e) {
        if (!cancelled) setTip(e instanceof Error ? e.message : '资料加载失败')
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  function persistSession(nextNick: string, nextAvatar: string) {
    if (!session) return
    const next = { ...session, nickname: nextNick, avatar: nextAvatar }
    saveLocalSession(next)
    setSession(next)
  }

  async function savePatch(patch: {
    nickname?: string
    avatarURL?: string
    selfSignature?: string
    gender?: number
    birthday?: number
    wananID?: string
  }) {
    setBusy(true)
    setTip('')
    try {
      const p = await updateProfile(patch)
      const nextNick = p.nickname || patch.nickname || nickname
      const nextAvatar = p.avatar_url || patch.avatarURL || avatar
      const nextSign = p.self_signature ?? patch.selfSignature ?? signature
      const nextGender = p.gender ?? patch.gender ?? gender
      const nextBirthday = p.birthday ?? patch.birthday ?? birthday
      const nextWanan = p.wanan_id || patch.wananID || wananId
      setNickname(nextNick)
      setAvatar(nextAvatar)
      setSignature(nextSign)
      setGender(Number(nextGender || 0))
      setBirthday(Number(nextBirthday || 0))
      setWananId(nextWanan)
      persistSession(nextNick, nextAvatar)
      await updateMyImProfile({
        nick: nextNick,
        avatar: nextAvatar,
        selfSignature: nextSign,
      }).catch(() => undefined)
      setEdit(null)
    } catch (e) {
      setTip(e instanceof Error ? e.message : '保存失败')
    } finally {
      setBusy(false)
    }
  }

  async function onPickAvatar(file?: File | null) {
    if (!file) return
    const uid = userId || session?.userId || ''
    if (!uid) {
      setTip('当前账号信息不完整，无法更换头像')
      alert('当前账号信息不完整，无法更换头像')
      return
    }
    setBusy(true)
    setTip('')
    try {
      const url = await uploadAvatarFile(file, { userId: uid })
      await savePatch({ avatarURL: url })
    } catch (e) {
      const msg = e instanceof Error ? e.message : '头像上传失败'
      setTip(msg)
      alert(msg)
    } finally {
      setBusy(false)
    }
  }

  function openEdit(key: EditKey) {
    if (key === 'nick') setDraft(nickname)
    else if (key === 'sign') setDraft(signature)
    else if (key === 'gender') setDraft(String(gender))
    else if (key === 'birthday') setDraft(birthdayToInput(birthday))
    else if (key === 'wanan') setDraft(wananId)
    setEdit(key)
  }

  async function confirmEdit() {
    if (edit === 'nick') {
      const v = draft.trim()
      if (!v) {
        setTip('昵称不能为空')
        return
      }
      await savePatch({ nickname: v })
      return
    }
    if (edit === 'sign') {
      await savePatch({ selfSignature: draft.trim() })
      return
    }
    if (edit === 'gender') {
      await savePatch({ gender: Number(draft) || 0 })
      return
    }
    if (edit === 'birthday') {
      const n = Number(draft.replace(/-/g, ''))
      await savePatch({ birthday: Number.isFinite(n) ? n : 0 })
      return
    }
    if (edit === 'wanan') {
      await savePatch({ wananID: draft.trim() })
    }
  }

  async function onLogout() {
    if (!confirm(t('logout_confirm'))) return
    setBusy(true)
    try {
      await logout()
      nav('/login', { replace: true })
    } finally {
      setBusy(false)
    }
  }

  async function onDelete() {
    if (!confirm(t('delete_account_confirm'))) return
    if (!confirm(t('delete_account_confirm2'))) return
    setBusy(true)
    setTip('')
    try {
      await deleteAccount()
      await logout()
      nav('/login', { replace: true })
    } catch (e) {
      setTip(e instanceof Error ? e.message : t('delete_account_fail'))
      setBusy(false)
    }
  }

  const name = nickname || userId || '—'

  return (
    <div className={styles.pane}>
      <div className={styles.hero}>
        <button
          type="button"
          className={styles.heroAvatar}
          disabled={busy}
          onClick={() => fileRef.current?.click()}
          title="更换头像"
        >
          {avatar ? <img src={avatar} alt="" /> : <span>{name.slice(0, 1)}</span>}
          <i className={styles.cam} aria-hidden>
            📷
          </i>
        </button>
        <strong>{loading ? '加载中…' : name}</strong>
      </div>

      <input
        ref={fileRef}
        type="file"
        accept="image/*"
        hidden
        onChange={(e) => {
          void onPickAvatar(e.target.files?.[0])
          e.target.value = ''
        }}
      />

      <div className={styles.card}>
        <button type="button" className={styles.row} onClick={() => openEdit('nick')}>
          <i className={`${styles.ico} ${styles.icoPurple}`}>🪪</i>
          <span>修改昵称</span>
          <em>{nickname || '未填写'}</em>
          <b>›</b>
        </button>
        <button type="button" className={styles.row} onClick={() => fileRef.current?.click()}>
          <i className={`${styles.ico} ${styles.icoBlue}`}>👤</i>
          <span>头像</span>
          <em>{busy ? '处理中…' : '更换'}</em>
          <b>›</b>
        </button>
      </div>

      <div className={styles.card}>
        <button type="button" className={styles.row} onClick={() => openEdit('sign')}>
          <i className={`${styles.ico} ${styles.icoPurple}`}>✎</i>
          <span>个性签名</span>
          <em>{signature || '未填写'}</em>
          <b>›</b>
        </button>
        <button type="button" className={styles.row} onClick={() => openEdit('gender')}>
          <i className={`${styles.ico} ${styles.icoBlue}`}>⚥</i>
          <span>性别</span>
          <em>{genderLabel(gender)}</em>
          <b>›</b>
        </button>
        <button type="button" className={styles.row} onClick={() => openEdit('birthday')}>
          <i className={`${styles.ico} ${styles.icoPeach}`}>🎂</i>
          <span>生日</span>
          <em>{birthdayLabel(birthday)}</em>
          <b>›</b>
        </button>
      </div>

      <div className={styles.card}>
        <div className={styles.row}>
          <i className={`${styles.ico} ${styles.icoPurple}`}>#</i>
          <span>用户ID</span>
          <em>{userId || '—'}</em>
        </div>
        <button type="button" className={styles.row} onClick={() => openEdit('wanan')}>
          <i className={`${styles.ico} ${styles.icoBlue}`}>@</i>
          <span>弥络蓝序号</span>
          <em>{wananId || '未填写'}</em>
          <b>›</b>
        </button>
      </div>

      <button type="button" className={styles.logout} disabled={busy} onClick={() => void onLogout()}>
        {t('logout')}
      </button>

      <div className={styles.section}>{t('danger_zone')}</div>
      <button type="button" className={styles.danger} disabled={busy} onClick={() => void onDelete()}>
        {t('delete_account')}
      </button>
      <p className={styles.hint}>{t('delete_account_hint')}</p>
      {tip ? <p className={styles.err}>{tip}</p> : null}

      {edit ? (
        <div className={styles.mask} onClick={() => !busy && setEdit(null)}>
          <div className={styles.dialog} onClick={(e) => e.stopPropagation()}>
            <h4>
              {edit === 'nick'
                ? '修改昵称'
                : edit === 'sign'
                  ? '个性签名'
                  : edit === 'gender'
                    ? '性别'
                    : edit === 'birthday'
                      ? '生日'
                      : '弥络蓝序号'}
            </h4>
            {edit === 'gender' ? (
              <div className={styles.seg}>
                {[
                  { v: '0', l: '未知' },
                  { v: '1', l: '男' },
                  { v: '2', l: '女' },
                ].map((x) => (
                  <button
                    key={x.v}
                    type="button"
                    className={draft === x.v ? styles.segOn : undefined}
                    onClick={() => setDraft(x.v)}
                  >
                    {x.l}
                  </button>
                ))}
              </div>
            ) : edit === 'birthday' ? (
              <input
                type="date"
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
              />
            ) : (
              <input
                value={draft}
                maxLength={edit === 'sign' ? 100 : 32}
                placeholder={edit === 'sign' ? '介绍一下自己' : ''}
                onChange={(e) => setDraft(e.target.value)}
              />
            )}
            <div className={styles.dialogActs}>
              <button type="button" disabled={busy} onClick={() => setEdit(null)}>
                取消
              </button>
              <button type="button" className={styles.ok} disabled={busy} onClick={() => void confirmEdit()}>
                {busy ? '保存中…' : '保存'}
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  )
}
