const { contextBridge, ipcRenderer } = require('electron')

contextBridge.exposeInMainWorld('desktop', {
  readAccountVault: () => ipcRenderer.invoke('account-vault-read'),
  writeAccountVault: (raw) => ipcRenderer.invoke('account-vault-write', raw),
  saveFile: (payload) => ipcRenderer.invoke('save-file', payload),
  saveFileAuto: (payload) => ipcRenderer.invoke('save-file-auto', payload),
  openPath: (filePath) => ipcRenderer.invoke('open-path', filePath),
  showInFolder: (filePath) => ipcRenderer.invoke('show-in-folder', filePath),
  putToCos: (payload) => ipcRenderer.invoke('cos-put', payload),
  captureScreen: () => ipcRenderer.invoke('capture-screen'),
})
