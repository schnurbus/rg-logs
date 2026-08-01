import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { Layout } from './components/Layout'
import { FightDetailPage } from './pages/FightDetailPage'
import { UploadDetailPage } from './pages/UploadDetailPage'
import { UploadPage } from './pages/UploadPage'
import { UploadsListPage } from './pages/UploadsListPage'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route index element={<UploadPage />} />
          <Route path="uploads" element={<UploadsListPage />} />
          <Route path="uploads/:uploadId" element={<UploadDetailPage />} />
          <Route path="fights/:fightId" element={<FightDetailPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}
