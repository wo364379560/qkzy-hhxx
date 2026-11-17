import styles from './PlaceholderPanel.module.css'

export function DiscoverPanel() {
  return (
    <div className={styles.pane}>
      <h3>发现</h3>
      <p className={styles.desc}>朋友圈等能力预留入口，后续可接动态流。</p>
      <div
        style={{
          marginTop: 16,
          borderRadius: 12,
          background: 'linear-gradient(135deg,#eef2ff,#f8fafc)',
          padding: 20,
          border: '1px solid var(--cn-border)',
        }}
      >
        <strong style={{ display: 'block', marginBottom: 6 }}>即将上线</strong>
        <span style={{ fontSize: 13, color: 'var(--cn-text-3)' }}>
          动态、附近、扫一扫等发现能力将在此扩展
        </span>
      </div>
    </div>
  )
}
