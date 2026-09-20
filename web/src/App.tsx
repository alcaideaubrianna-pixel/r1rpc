import type { ReactNode } from 'react'
import { Routes, Route, Navigate, useLocation } from 'react-router-dom'
import { auth } from './auth'
import Shell from './Shell'
import LoginPage from './pages/LoginPage'
import OverviewPage from './pages/OverviewPage'
import GroupsPage from './pages/GroupsPage'
import DevicesPage from './pages/DevicesPage'
import ClientsPage from './pages/ClientsPage'
import RequestsPage from './pages/RequestsPage'
import InvokePage from './pages/InvokePage'
import UsersPage from './pages/UsersPage'
import FilesPage from './pages/FilesPage'
import ImageSearchPage from './pages/ImageSearchPage'
import ImageSearchDetailPage from './pages/ImageSearchDetailPage'
import SearchTaskDetailPage from './pages/SearchTaskDetailPage'
import DataSourcesPage from './pages/DataSourcesPage'

function RequireAuth({ children }: { children: ReactNode }) {
  const loc = useLocation()
  if (!auth.token) return <Navigate to="/login" replace state={{ from: loc.pathname }} />
  return <>{children}</>
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        element={
          <RequireAuth>
            <Shell />
          </RequireAuth>
        }
      >
        <Route index element={<Navigate to="/overview" replace />} />
        <Route path="overview" element={<OverviewPage />} />
        <Route path="groups" element={<GroupsPage />} />
        <Route path="devices" element={<DevicesPage />} />
        <Route path="clients" element={<ClientsPage />} />
        <Route path="requests" element={<RequestsPage />} />
        <Route path="invoke" element={<InvokePage />} />
        <Route path="files" element={<FilesPage />} />
        <Route path="image-search" element={<ImageSearchPage />} />
        <Route path="image-search/:id" element={<SearchTaskDetailPage />} />
        <Route path="image-search/results/:id" element={<ImageSearchDetailPage />} />
        <Route path="data-sources" element={<DataSourcesPage />} />
        <Route path="users" element={<UsersPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/overview" replace />} />
    </Routes>
  )
}
