import styles from './PrivacyDialog.module.css'

type Props = {
  open: boolean
  onAgree: () => void
  onDisagree: () => void
}

export function PrivacyDialog({ open, onAgree, onDisagree }: Props) {
  if (!open) return null
  return (
    <div className={styles.mask}>
      <div className={styles.dialog} role="dialog" aria-modal>
        <h3>用户协议与隐私政策</h3>
        <p>
          请阅读并同意《用户协议》和《隐私政策》后使用本应用。点击「暂不使用」将退出；下次打开未同意仍会弹出。
        </p>
        <div className={styles.actions}>
          <button type="button" className={styles.ghost} onClick={onDisagree}>
            暂不使用
          </button>
          <button type="button" className={styles.primary} onClick={onAgree}>
            同意
          </button>
        </div>
      </div>
    </div>
  )
}
