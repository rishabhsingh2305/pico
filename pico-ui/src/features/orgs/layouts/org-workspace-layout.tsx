import { useCallback, useEffect } from "react"
import {
  Link,
  Outlet,
  useNavigate,
  useOutletContext,
  useParams,
} from "react-router"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import {
  BarChart3Icon,
  Building2Icon,
  CircleAlertIcon,
  FolderIcon,
  LinkIcon,
  ReceiptTextIcon,
  Share2Icon,
} from "lucide-react"

import { queryKeys } from "@/app/query-keys"
import { AppShell, type NavigationGroup } from "@/components/app-shell"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Spinner } from "@/components/ui/spinner"
import { useAuth, useSession } from "@/features/auth/auth-provider"
import { isSessionInvalidError } from "@/features/auth/session"
import { getOrg } from "@/features/orgs/api"
import { orgErrorMessage } from "@/features/orgs/error-message"
import type { Org } from "@/features/orgs/types"
import { PublicShareBadge } from "@/features/sharing/components/public-share-badge"
import { ShareDialog } from "@/features/sharing/components/share-dialog"
import { getShareStatus } from "@/features/sharing/api"
import { ApiError } from "@/lib/api-client"
import { hasFeatureRole } from "@/lib/roles"

export type OrgWorkspaceContext = {
  org: Org
  roles: string[]
  isOwner: boolean
  canShare: boolean
  canViewShares: boolean
  canViewLinks: boolean
  canViewAnalytics: boolean
  canManageAnalytics: boolean
  sessionInvalid: () => void
}

const fallbackNavigation: NavigationGroup[] = [
  {
    label: "Workspace",
    items: [
      {
        to: "/orgs",
        label: "Organizations",
        index: "01",
        icon: Building2Icon,
        isActive: () => true,
      },
    ],
  },
]

export function useOrgWorkspace(): OrgWorkspaceContext {
  return useOutletContext<OrgWorkspaceContext>()
}

export function OrgWorkspaceLayout() {
  const session = useSession()
  const { orgId = "" } = useParams<{ orgId: string }>()
  const navigate = useNavigate()
  const { logout } = useAuth()
  const queryClient = useQueryClient()

  const orgQuery = useQuery({
    queryKey: queryKeys.org(session.userId, orgId),
    queryFn: () => getOrg(orgId),
    enabled: orgId !== "",
  })

  const roles = session.orgs?.[orgId] ?? []
  const isOwner = orgQuery.data?.userId === session.userId
  const canShare = isOwner || roles.includes("share.admin")
  const canViewShares = isOwner || hasFeatureRole(roles, "share")
  const canViewLinks = isOwner || hasFeatureRole(roles, "link")
  const canViewAnalytics = isOwner || hasFeatureRole(roles, "analytics")
  const canManageAnalytics =
    isOwner ||
    roles.includes("analytics.editor") ||
    roles.includes("analytics.admin")

  const sessionInvalid = useCallback(() => {
    void logout()
    navigate("/login", { replace: true })
  }, [logout, navigate])

  useEffect(() => {
    if (orgQuery.error && isSessionInvalidError(orgQuery.error)) {
      sessionInvalid()
    }
  }, [orgQuery.error, sessionInvalid])

  const org = orgQuery.data ?? null

  const orgShareStatusQuery = useQuery({
    queryKey: queryKeys.shareStatus(orgId, "org", [orgId]),
    queryFn: () => getShareStatus(orgId, "org", [orgId]),
    enabled: org !== null && canViewShares && orgId !== "",
  })

  const invalidateShareStatus = useCallback(() => {
    void queryClient.invalidateQueries({
      queryKey: queryKeys.shareStatus(orgId, "org", [orgId]),
    })
  }, [orgId, queryClient])

  if (orgQuery.isPending) {
    return (
      <AppShell session={session} navigation={fallbackNavigation}>
        <Card>
          <CardContent className="flex items-center justify-center gap-2 py-12 text-sm text-muted-foreground">
            <Spinner />
            Loading organization...
          </CardContent>
        </Card>
      </AppShell>
    )
  }

  if (orgQuery.isError || !org) {
    return (
      <AppShell session={session} navigation={fallbackNavigation}>
        <Card>
          <CardContent className="flex flex-col gap-4 py-6">
            <Alert variant="destructive">
              <CircleAlertIcon aria-hidden="true" />
              <AlertDescription>
                {orgQuery.isError
                  ? orgWorkspaceErrorMessage(orgQuery.error)
                  : "Organization not found."}
              </AlertDescription>
            </Alert>
            <div className="flex gap-2">
              <Button
                type="button"
                variant="outline"
                onClick={() => void orgQuery.refetch()}
              >
                Try again
              </Button>
              <Link to="/orgs" className={buttonVariants({ variant: "ghost" })}>
                Back to organizations
              </Link>
            </div>
          </CardContent>
        </Card>
      </AppShell>
    )
  }

  const context: OrgWorkspaceContext = {
    org,
    roles,
    isOwner,
    canShare,
    canViewShares,
    canViewLinks,
    canViewAnalytics,
    canManageAnalytics,
    sessionInvalid,
  }
  const orgShared = (orgShareStatusQuery.data ?? []).length > 0
  const navigation = buildOrgNavigation(
    orgId,
    canViewLinks,
    canViewAnalytics,
    canViewShares
  )

  return (
    <AppShell
      session={session}
      navigation={navigation}
      sidebarContext={
        <div className="flex flex-col gap-3">
          <Link
            to="/orgs"
            className="text-label text-sidebar-foreground/60 uppercase underline-offset-4 hover:text-sidebar-foreground hover:underline"
          >
            ← All organizations
          </Link>
          <div className="min-w-0">
            <p className="truncate font-display text-lg font-semibold tracking-tight">
              {org.name}
            </p>
            {org.tags.length > 0 ? (
              <p className="mt-1 truncate text-xs text-sidebar-foreground/55">
                {org.tags.join(", ")}
              </p>
            ) : null}
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant={isOwner ? "default" : "secondary"}>
              {isOwner ? "Owner" : "Member"}
            </Badge>
            {orgShared ? <PublicShareBadge /> : null}
            {canShare ? (
              <ShareDialog
                orgId={orgId}
                resourceType="org"
                resourceName={org.name}
                onChanged={invalidateShareStatus}
                onSessionInvalid={sessionInvalid}
              />
            ) : null}
          </div>
        </div>
      }
    >
      <Outlet context={context} />
    </AppShell>
  )
}

function buildOrgNavigation(
  orgId: string,
  canViewLinks: boolean,
  canViewAnalytics: boolean,
  canViewShares: boolean
): NavigationGroup[] {
  const workspaceItems = [
    {
      to: `/orgs/${orgId}/projects`,
      label: "Projects",
      index: "01",
      icon: FolderIcon,
      isActive: (pathname: string) =>
        pathname.startsWith(`/orgs/${orgId}/projects`),
    },
    {
      to: `/orgs/${orgId}/expenses`,
      label: "Expenses",
      index: "02",
      icon: ReceiptTextIcon,
      isActive: (pathname: string) =>
        pathname.startsWith(`/orgs/${orgId}/expenses`),
    },
  ]

  if (canViewLinks) {
    workspaceItems.push({
      to: `/orgs/${orgId}/links`,
      label: "Links",
      index: "03",
      icon: LinkIcon,
      isActive: (pathname: string) =>
        pathname.startsWith(`/orgs/${orgId}/links`),
    })
  }

  if (canViewAnalytics) {
    workspaceItems.push({
      to: `/orgs/${orgId}/analytics`,
      label: "Analytics",
      index: "04",
      icon: BarChart3Icon,
      isActive: (pathname: string) =>
        pathname.startsWith(`/orgs/${orgId}/analytics`),
    })
  }

  const groups: NavigationGroup[] = [
    {
      label: "Workspace",
      items: workspaceItems,
    },
  ]

  if (canViewShares) {
    groups.push({
      label: "Access",
      items: [
        {
          to: `/orgs/${orgId}/sharing`,
          label: "Sharing",
          index: "01",
          icon: Share2Icon,
          isActive: (pathname: string) =>
            pathname.startsWith(`/orgs/${orgId}/sharing`),
        },
      ],
    })
  }

  return groups
}

function orgWorkspaceErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status === 403) {
      return "You no longer have access to this organization."
    }
    if (error.status === 404) {
      return "Organization not found."
    }
  }
  return orgErrorMessage(error)
}
