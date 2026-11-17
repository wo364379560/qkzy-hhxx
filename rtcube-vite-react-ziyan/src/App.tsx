import { Navigate, Route, Routes } from 'react-router-dom'
import { getLocalSession } from '@/api/session'
import { I18nProvider } from '@/i18n/I18nProvider'
import { ChatShell } from '@/pages/ChatShell'
import { LoginPage } from '@/pages/LoginPage'
import { RegisterPhonePage } from '@/pages/RegisterPhonePage'
import { RegisterSelectPage } from '@/pages/RegisterSelectPage'
import { RegisterUsernamePage } from '@/pages/RegisterUsernamePage'
import { AppStoreProvider } from '@/store/AppStore'

function HomeRedirect() {
  const s = getLocalSession()
  return <Navigate to={s?.userId && s.userSig ? '/chat' : '/login'} replace />
}

export default function App() {
  return (
    <I18nProvider>
      <AppStoreProvider>
        <Routes>
          <Route path="/" element={<HomeRedirect />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterSelectPage />} />
          <Route path="/register/phone" element={<RegisterPhonePage />} />
          <Route path="/register/username" element={<RegisterUsernamePage />} />
          <Route path="/chat" element={<ChatShell />} />
          <Route path="*" element={<HomeRedirect />} />
        </Routes>
      </AppStoreProvider>
    </I18nProvider>
  )
}
