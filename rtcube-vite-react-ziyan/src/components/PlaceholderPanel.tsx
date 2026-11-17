import styles from './PlaceholderPanel.module.css'

export function PlaceholderPanel({ title, desc }: { title: string; desc: string }) {
  return (
    <div className={styles.pane}>
      <h3>{title}</h3>
      <p className={styles.desc}>{desc}</p>
    </div>
  )
}
