const { app, BrowserWindow, shell, ipcMain, dialog, session, desktopCapturer, screen, safeStorage } = require('electron')
const fs = require('fs')
const path = require('path')
const http = require('http')
const https = require('https')

function createWindow() {
  const win = new BrowserWindow({
    width: 1280,
    height: 820,
    minWidth: 960,
    minHeight: 640,
    webPreferences: {
      preload: path.join(__dirname, 'preload.cjs'),
      nodeIntegration: false,
      contextIsolation: true,
    },
    show: false,
  })

  if (process.env.NODE_ENV === 'development') {
    win.loadURL('http://localhost:5175')
    win.webContents.openDevTools({ mode: 'detach' })
  } else {
    win.loadFile(path.join(__dirname, '../dist/index.html'))
  }

  win.once('ready-to-show', () => win.show())
  win.webContents.on('did-finish-load', () => {
    win.webContents.executeJavaScript('window.__ELECTRON__ = true')
  })
  win.webContents.setWindowOpenHandler(() => ({ action: 'deny' }))
}

function safeFileName(name) {
  return String(name || 'download').replace(/[\\/:*?"<>|]/g, '_').slice(0, 120)
}

function uniquePath(dir, name) {
  const base = safeFileName(name)
  let dest = path.join(dir, base)
  if (!fs.existsSync(dest)) return dest
  const ext = path.extname(base)
  const stem = path.basename(base, ext)
  for (let i = 1; i < 200; i++) {
    dest = path.join(dir, `${stem}(${i})${ext}`)
    if (!fs.existsSync(dest)) return dest
  }
  return path.join(dir, `${stem}_${Date.now()}${ext}`)
}

function accountVaultPath() {
  return path.join(app.getPath('userData'), 'account-vault.bin')
}

ipcMain.handle('account-vault-read', async () => {
  const filePath = accountVaultPath()
  if (!fs.existsSync(filePath)) return ''
  if (!safeStorage.isEncryptionAvailable()) {
    throw new Error('系统安全存储不可用，无法读取账号列表')
  }
  return safeStorage.decryptString(fs.readFileSync(filePath))
})

ipcMain.handle('account-vault-write', async (_event, raw) => {
  if (typeof raw !== 'string' || Buffer.byteLength(raw, 'utf8') > 1024 * 1024) {
    throw new Error('账号数据无效')
  }
  if (!safeStorage.isEncryptionAvailable()) {
    throw new Error('系统安全存储不可用，无法保存账号列表')
  }
  const filePath = accountVaultPath()
  const tempPath = `${filePath}.${process.pid}.tmp`
  fs.mkdirSync(path.dirname(filePath), { recursive: true })
  fs.writeFileSync(tempPath, safeStorage.encryptString(raw), { mode: 0o600 })
  fs.renameSync(tempPath, filePath)
  return true
})

ipcMain.handle('save-file', async (event, payload) => {
  const name = String(payload?.name || 'download')
  const buffer = payload?.buffer
  if (!buffer) return null
  const win = BrowserWindow.fromWebContents(event.sender)
  const res = await dialog.showSaveDialog(win || undefined, {
    defaultPath: name,
  })
  if (res.canceled || !res.filePath) return null
  fs.writeFileSync(res.filePath, Buffer.from(buffer))
  return res.filePath
})

ipcMain.handle('save-file-auto', async (_event, payload) => {
  const buffer = payload?.buffer
  if (!buffer) return null
  const dir = path.join(app.getPath('downloads'), 'Messaging - IM')
  fs.mkdirSync(dir, { recursive: true })
  const dest = uniquePath(dir, payload?.name || 'download')
  fs.writeFileSync(dest, Buffer.from(buffer))
  return dest
})

ipcMain.handle('open-path', async (_event, filePath) => {
  if (!filePath) return
  await shell.openPath(String(filePath))
})

ipcMain.handle('show-in-folder', async (_event, filePath) => {
  if (!filePath) return
  shell.showItemInFolder(String(filePath))
})

ipcMain.handle('capture-screen', async (event) => {
  const win = BrowserWindow.fromWebContents(event.sender)
  const display = win ? screen.getDisplayMatching(win.getBounds()) : screen.getPrimaryDisplay()
  const scale = display.scaleFactor || 1
  const width = Math.round(display.size.width * scale)
  const height = Math.round(display.size.height * scale)
  const wasVisible = Boolean(win && win.isVisible())
  if (win && wasVisible) win.hide()
  await new Promise((resolve) => setTimeout(resolve, 60))
  try {
    const sources = await desktopCapturer.getSources({
      types: ['screen'],
      thumbnailSize: { width, height },
    })
    const id = String(display.id)
    const source =
      sources.find((s) => String(s.display_id) === id) ||
      sources.find((s) => s.id.includes('screen')) ||
      sources[0]
    if (!source?.thumbnail) throw new Error('无法截取屏幕')
    return source.thumbnail.toPNG()
  } finally {
    if (win && !win.isDestroyed() && wasVisible) win.show()
  }
})

function isAllowedCosHost(hostname) {
  return /(\.myqcloud\.com|\.tencentcos\.cn)$/i.test(String(hostname || ''))
}

ipcMain.handle('cos-put', async (_event, payload) => {
  const putUrl = String(payload?.url || '')
  const filePath = String(payload?.filePath || '')
  const contentType = String(payload?.contentType || 'application/octet-stream')
  if (!putUrl || !filePath) throw new Error('云端上传参数不完整')
  let target
  try {
    target = new URL(putUrl)
  } catch {
    throw new Error('云端上传地址无效')
  }
  if (!isAllowedCosHost(target.hostname)) throw new Error('云端上传地址无效')
  if (!fs.existsSync(filePath)) throw new Error('本地文件不存在')
  const size = fs.statSync(filePath).size
  if (size <= 0) throw new Error('本地文件为空')
  const lib = target.protocol === 'https:' ? https : http
  await new Promise((resolve, reject) => {
    const req = lib.request(
      {
        method: 'PUT',
        hostname: target.hostname,
        port: target.port || undefined,
        path: `${target.pathname}${target.search}`,
        headers: {
          'Content-Type': contentType,
          'Content-Length': size,
        },
      },
      (res) => {
        const chunks = []
        res.on('data', (c) => chunks.push(c))
        res.on('end', () => {
          if (res.statusCode >= 200 && res.statusCode < 300) {
            resolve()
            return
          }
          const body = Buffer.concat(chunks).toString('utf8').slice(0, 200)
          reject(new Error(`云端上传失败（${res.statusCode}）${body ? ` ${body}` : ''}`))
        })
      },
    )
    req.on('error', (err) => reject(err))
    fs.createReadStream(filePath).pipe(req)
  })
  return true
})

app.whenReady().then(() => {
  session.defaultSession.setPermissionRequestHandler((_wc, permission, callback) => {
    if (permission === 'media' || permission === 'microphone' || permission === 'camera' || permission === 'audioCapture' || permission === 'videoCapture') {
      callback(true)
      return
    }
    callback(false)
  })
  session.defaultSession.setPermissionCheckHandler((_wc, permission) => {
    return (
      permission === 'media' ||
      permission === 'microphone' ||
      permission === 'camera' ||
      permission === 'audioCapture' ||
      permission === 'videoCapture'
    )
  })
  createWindow()
  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) createWindow()
  })
})

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') app.quit()
})
