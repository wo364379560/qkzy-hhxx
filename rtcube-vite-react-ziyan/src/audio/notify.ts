export function requestNotifyPermission() {
  if (typeof Notification === 'undefined') return
  if (Notification.permission === 'default') {
    void Notification.requestPermission().catch(() => undefined)
  }
}

export function showAppNotification(opts: {
  title: string
  body?: string
  tag?: string
  force?: boolean
}) {
  if (typeof Notification === 'undefined') return
  if (Notification.permission !== 'granted') return
  if (!opts.force && !document.hidden) return
  try {
    const n = new Notification(opts.title || '新消息', {
      body: opts.body || '',
      tag: opts.tag || 'cn-im',
      silent: true,
    })
    n.onclick = () => {
      try {
        window.focus()
      } catch {
        /* ignore */
      }
      n.close()
    }
  } catch {
    /* ignore */
  }
}
