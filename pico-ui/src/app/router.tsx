import { Navigate, Route, Routes } from "react-router"

import { ApiDocsPage } from "@/features/api-docs/pages/api-docs-page"
import { getSession } from "@/features/auth/session"
import { SignUpPage } from "@/features/auth/pages/sign-up-page"
import { VerifyEmailPage } from "@/features/auth/pages/verify-email-page"
import { OrgDashboardPage } from "@/features/dashboard/pages/dashboard-page"
import { StyleTilePage } from "@/features/design-system/pages/style-tile-page"
import { OrgDetailPage } from "@/features/orgs/pages/org-detail-page"
import { OrgsPage } from "@/features/orgs/pages/orgs-page"
import { ProjectDetailPage } from "@/features/orgs/pages/project-detail-page"

function HomeRedirect() {
  return <Navigate to={getSession() ? "/orgs" : "/signup"} replace />
}

export function AppRouter() {
  return (
    <Routes>
      <Route path="/signup" element={<SignUpPage />} />
      <Route path="/verify-email" element={<VerifyEmailPage />} />
      <Route path="/api-docs" element={<ApiDocsPage />} />
      <Route path="/design" element={<StyleTilePage />} />
      <Route path="/orgs" element={<OrgsPage />} />
      <Route path="/orgs/:orgId/dashboard" element={<OrgDashboardPage />} />
      <Route path="/orgs/:orgId" element={<OrgDetailPage />} />
      <Route
        path="/orgs/:orgId/projects/:projectId"
        element={<ProjectDetailPage />}
      />
      <Route path="/" element={<HomeRedirect />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
