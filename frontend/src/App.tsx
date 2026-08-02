import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { AuthProvider } from './auth/AuthProvider'
import { Layout } from './components/Layout'
import { AuthCallbackPage } from './pages/AuthCallbackPage'
import { FightDetailPage } from './pages/FightDetailPage'
import { FightEventsPage } from './pages/FightEventsPage'
import { LoginPage } from './pages/LoginPage'
import { UploadDetailPage } from './pages/UploadDetailPage'
import { UploadPage } from './pages/UploadPage'
import { UploadsListPage } from './pages/UploadsListPage'

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route element={<Layout />}>
            <Route index element={<UploadPage />} />
            <Route path="login" element={<LoginPage />} />
            <Route path="auth/callback" element={<AuthCallbackPage />} />
            <Route path="uploads" element={<UploadsListPage />} />
            <Route path="uploads/:uploadId" element={<UploadDetailPage />} />
            <Route path="fights/:fightId" element={<FightDetailPage />} />
            <Route
              path="fights/:fightId/events"
              element={<FightEventsPage />}
            />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  )
}
