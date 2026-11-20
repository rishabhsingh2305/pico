import { Navigate, Route, Routes, useParams } from "react-router"

import { ApiConsolePage } from "@/features/developer/pages/api-console-page"
import { ApiDocsPage } from "@/features/api-docs/pages/api-docs-page"
import { RequireSession } from "@/features/auth/auth-provider"
import { LoginPage } from "@/features/auth/pages/login-page"
import { SignUpPage } from "@/features/auth/pages/sign-up-page"
import { VerifyEmailPage } from "@/features/auth/pages/verify-email-page"
import { StyleTilePage } from "@/features/design-system/pages/style-tile-page"
import { LinksPage } from "@/features/links/pages/links-page"
import { OrgWorkspaceLayout } from "@/features/orgs/layouts/org-workspace-layout"
import { OrgExpensesPage } from "@/features/orgs/pages/org-expenses-page"
import { OrgProjectsPage } from "@/features/orgs/pages/org-projects-page"
import { OrgsPage } from "@/features/orgs/pages/orgs-page"
import { ProjectDetailPage } from "@/features/orgs/pages/project-detail-page"
import { SharingPage } from "@/features/sharing/pages/sharing-page"
import { SharedPage } from "@/features/sharing/pages/shared-page"
import { WorkspaceLayout } from "@/app/workspace-layout"

function OrgExpensesRedirect() {
  const { orgId = "" } = useParams<{ orgId: string }>()
  return <Navigate to={`/orgs/${orgId}/expenses`} replace />
}

export function AppRouter() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/signup" element={<SignUpPage />} />
      <Route path="/verify-email" element={<VerifyEmailPage />} />
      <Route path="/api-docs" element={<ApiDocsPage />} />
      <Route path="/design" element={<StyleTilePage />} />
      <Route path="/shared" element={<SharedPage />} />
      <Route path="/orgs/:orgId/dashboard" element={<OrgExpensesRedirect />} />

      <Route element={<RequireSession />}>
        <Route path="/" element={<Navigate to="/orgs" replace />} />
        <Route element={<WorkspaceLayout />}>
          <Route path="/orgs" element={<OrgsPage />} />
          <Route path="/api" element={<ApiConsolePage />} />
        </Route>
        <Route path="/orgs/:orgId" element={<OrgWorkspaceLayout />}>
          <Route index element={<Navigate to="projects" replace />} />
          <Route path="projects" element={<OrgProjectsPage />} />
          <Route path="projects/:projectId" element={<ProjectDetailPage />} />
          <Route path="expenses" element={<OrgExpensesPage />} />
          <Route path="links" element={<LinksPage />} />
          <Route path="sharing" element={<SharingPage />} />
        </Route>
      </Route>

      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
