import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter, HashRouter } from 'react-router-dom'
import App from './App'
import { initializeRuntimeEndpoints } from './config/runtime'
import './styles/global.css'

const Router = window.__ELECTRON__ ? HashRouter : BrowserRouter

async function start() {
  await initializeRuntimeEndpoints()
  ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
      <Router>
        <App />
      </Router>
    </React.StrictMode>,
  )
}

void start()
