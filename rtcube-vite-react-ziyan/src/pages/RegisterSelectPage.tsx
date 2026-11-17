import { Link, useNavigate } from 'react-router-dom'
import { APP_NAME } from '@/config/runtime'
import styles from './LoginPage.module.css'

export function RegisterSelectPage() {
  const nav = useNavigate()

  return (
    <div className={styles.page}>
      <div className={styles.card}>
        <div className={styles.brand}>
          <div className={styles.logo}>{APP_NAME.slice(0, 1)}</div>
          <h1>创建账号</h1>
        </div>

        <div className={styles.form}>
          <button
            type="button"
            className={styles.submit}
            onClick={() => nav('/register/phone')}
          >
            手机号注册
          </button>
          <button
            type="button"
            className={styles.secondary}
            onClick={() => nav('/register/username')}
          >
            用户名注册
          </button>
          <p className={styles.footer}>
            已有账号？<Link to="/login">立即登录</Link>
          </p>
        </div>
      </div>
    </div>
  )
}
