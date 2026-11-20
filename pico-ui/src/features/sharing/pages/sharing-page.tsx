import { useEffect, useState } from "react"
import { useNavigate, useParams } from "react-router"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import {
  BanIcon,
  ChevronRightIcon,
  CircleAlertIcon,
  ExternalLinkIcon,
  PlusIcon,
  RefreshCwIcon,
  SearchIcon,
  Share2Icon,
} from "lucide-react"

import { queryKeys } from "@/app/query-keys"
import { PageHeader } from "@/components/page-header"
import { SelectField } from "@/components/select-field"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { Spinner } from "@/components/ui/spinner"
import { useSession } from "@/features/auth/auth-provider"
import { isSessionInvalidError, type Session } from "@/features/auth/session"
import { useOrgWorkspace } from "@/features/orgs/layouts/org-workspace-layout"
import { listProjects } from "@/features/projects/api"
import { listOrgShares, revokeShare } from "@/features/sharing/api"
import { CreateShareDialog } from "@/features/sharing/components/create-share-dialog"
import { shareErrorMessage } from "@/features/sharing/error-message"
import {
  accessSummary,
  groupScopes,
  scopeLabel,
} from "@/features/sharing/permissions"
import type {
  ShareCredential,
  ShareResourceType,
} from "@/features/sharing/types"
import { usePageTitle } from "@/hooks/use-page-title"
import { cn } from "cn"

type ResourceFilter = ShareResourceType | "all"
type StatusFilter = "all" | "active" | "expired" | "revoked"

type CreateTarget = {
  resourceType: "org" | "project"
  projectID?: string
}

export function SharingPage() {
  const session = useSession()
  return <SharingView session={session} />
}

function SharingView({ session }: { session: Session }) {
  const { orgId = "" } = useParams<{ orgId: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { org, canShare, canViewShares, sessionInvalid } = useOrgWorkspace()

  const [resourceType, setResourceType] = useState<ResourceFilter>("all")
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all")
  const [query, setQuery] = useState("")
  const [selected, setSelected] = useState<ShareCredential | null>(null)
  const [createTarget, setCreateTarget] = useState<CreateTarget | null>(null)
  const [revoking, setRevoking] = useState<ShareCredential | null>(null)
  const [revokingBusy, setRevokingBusy] = useState(false)
  const [actionError, setActionError] = useState("")

  usePageTitle(`${org.name} sharing · Pico`)

  const sharesQuery = useQuery({
    queryKey: queryKeys.orgShares(orgId, resourceType, false),
    queryFn: () => listOrgShares(orgId, { resourceType }),
    enabled: canViewShares && orgId !== "",
  })
  const projectsQuery = useQuery({
    queryKey: queryKeys.projects(session.userId, orgId, []),
    queryFn: () => listProjects(orgId),
    enabled: canViewShares && orgId !== "",
  })

  useEffect(() => {
    const error = sharesQuery.error ?? projectsQuery.error
    if (error && isSessionInvalidError(error)) {
      sessionInvalid()
    }
  }, [sharesQuery.error, projectsQuery.error, sessionInvalid])

  const shares = sharesQuery.data ?? []
  const projects = projectsQuery.data ?? []
  const projectNames = Object.fromEntries(
    projects.map((project) => [project.id, project.name])
  )
  const counts = {
    all: shares.length,
    active: shares.filter((share) => share.status === "active").length,
    expired: shares.filter((share) => share.status === "expired").length,
    revoked: shares.filter((share) => share.status === "revoked").length,
  }
  const normalizedQuery = query.trim().toLowerCase()
  const visibleShares = shares.filter((share) => {
    if (statusFilter !== "all" && share.status !== statusFilter) {
      return false
    }
    if (normalizedQuery === "") {
      return true
    }
    const label = resourceLabel(share, org.name, projectNames)
    const name = share.name?.trim().toLowerCase() ?? ""
    return (
      name.includes(normalizedQuery) ||
      label.toLowerCase().includes(normalizedQuery)
    )
  })

  function invalidate() {
    void queryClient.invalidateQueries({
      queryKey: queryKeys.orgShares(orgId, resourceType, false),
    })
  }

  function openResource(share: ShareCredential) {
    if (share.resourceType === "org") {
      navigate(`/orgs/${orgId}`)
      return
    }
    const projectID =
      share.resourceType === "project" ? share.resourceId : share.projectId
    if (projectID) {
      navigate(`/orgs/${orgId}/projects/${projectID}`)
    }
  }

  function startReplacement(share: ShareCredential) {
    if (share.resourceType === "org") {
      setCreateTarget({ resourceType: "org" })
    } else if (share.resourceType === "project") {
      setCreateTarget({
        resourceType: "project",
        projectID: share.projectId ?? share.resourceId,
      })
    }
    setSelected(null)
  }

  async function handleRevoke() {
    if (!revoking) {
      return
    }
    setRevokingBusy(true)
    setActionError("")
    try {
      await revokeShare(orgId, revoking.id)
      setRevoking(null)
      setSelected(null)
      invalidate()
    } catch (error) {
      if (isSessionInvalidError(error)) {
        sessionInvalid()
        return
      }
      setRevoking(null)
      setActionError(shareErrorMessage(error))
    } finally {
      setRevokingBusy(false)
    }
  }

  function clearFilters() {
    setQuery("")
    setStatusFilter("all")
    setResourceType("all")
  }

  if (!canViewShares) {
    return (
      <Card>
        <CardContent className="flex flex-col gap-4 py-8">
          <Alert>
            <CircleAlertIcon aria-hidden="true" />
            <AlertDescription>
              You do not have access to sharing in this organization.
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Access"
        title="Sharing"
        description={`Read-only links for ${org.name}. URLs are shown once when created, so this page lists their access, status, and expiry.`}
        actions={
          canShare ? (
            <Button
              type="button"
              className="h-10 px-4 text-xs font-semibold tracking-[0.14em] uppercase"
              onClick={() => setCreateTarget({ resourceType: "org" })}
            >
              <PlusIcon data-icon="inline-start" />
              New share link
            </Button>
          ) : null
        }
      />

      <div className="flex flex-col gap-4">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
          <div className="relative min-w-0 flex-1">
            <SearchIcon
              className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground"
              aria-hidden="true"
            />
            <Input
              aria-label="Search share links"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Search name or resource"
              className="pl-8"
            />
          </div>
          <SelectField
            aria-label="Filter by resource"
            value={resourceType}
            onValueChange={(value) => setResourceType(value as ResourceFilter)}
            options={[
              { value: "all", label: "All resources" },
              { value: "org", label: "Organization" },
              { value: "project", label: "Projects" },
              { value: "expense", label: "Expenses" },
            ]}
            className="sm:w-48"
          />
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => void sharesQuery.refetch()}
            disabled={sharesQuery.isFetching}
          >
            <RefreshCwIcon data-icon="inline-start" />
            Refresh
          </Button>
        </div>

        <div
          className="flex gap-1 overflow-x-auto border-b border-border"
          role="tablist"
          aria-label="Share status"
        >
          {(
            [
              ["all", "All"],
              ["active", "Active"],
              ["expired", "Expired"],
              ["revoked", "Revoked"],
            ] as const
          ).map(([value, label]) => (
            <button
              key={value}
              type="button"
              role="tab"
              aria-selected={statusFilter === value}
              onClick={() => setStatusFilter(value)}
              className={cn(
                "shrink-0 border-b-2 px-3 py-2.5 text-sm font-medium transition-colors focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none",
                statusFilter === value
                  ? "border-primary text-foreground"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              )}
            >
              {label} {counts[value]}
            </button>
          ))}
        </div>
      </div>

      {actionError ? (
        <Alert variant="destructive">
          <CircleAlertIcon aria-hidden="true" />
          <AlertDescription>{actionError}</AlertDescription>
        </Alert>
      ) : null}

      {sharesQuery.isPending ? (
        <Card>
          <CardContent className="flex items-center justify-center gap-2 py-10 text-sm text-muted-foreground">
            <Spinner />
            Loading share links...
          </CardContent>
        </Card>
      ) : null}

      {sharesQuery.isError ? (
        <Card>
          <CardContent className="flex flex-col gap-4 py-6">
            <Alert variant="destructive">
              <CircleAlertIcon aria-hidden="true" />
              <AlertDescription>
                {shareErrorMessage(sharesQuery.error)}
              </AlertDescription>
            </Alert>
            <Button
              type="button"
              variant="outline"
              className="w-fit"
              onClick={() => void sharesQuery.refetch()}
            >
              Try again
            </Button>
          </CardContent>
        </Card>
      ) : null}

      {sharesQuery.isSuccess && shares.length === 0 ? (
        <div className="flex flex-col items-center gap-3 border-y border-border py-14 text-center">
          <Share2Icon
            className="size-5 text-muted-foreground"
            aria-hidden="true"
          />
          <p className="font-display text-2xl font-semibold tracking-tight">
            No share links yet
          </p>
          <p className="max-w-sm text-sm text-muted-foreground">
            Create a read-only link for this organization or one of its
            projects.
          </p>
          {canShare ? (
            <Button
              type="button"
              onClick={() => setCreateTarget({ resourceType: "org" })}
            >
              <PlusIcon data-icon="inline-start" />
              New share link
            </Button>
          ) : null}
        </div>
      ) : null}

      {shares.length > 0 && visibleShares.length === 0 ? (
        <div className="flex flex-col items-center gap-3 border-y border-border py-14 text-center">
          <SearchIcon
            className="size-5 text-muted-foreground"
            aria-hidden="true"
          />
          <p className="font-display text-2xl font-semibold tracking-tight">
            No matching share links
          </p>
          <p className="max-w-sm text-sm text-muted-foreground">
            Try another search, status, or resource filter.
          </p>
          <Button type="button" variant="outline" onClick={clearFilters}>
            Clear filters
          </Button>
        </div>
      ) : null}

      {visibleShares.length > 0 ? (
        <ul className="divide-y divide-border border-y border-border">
          {visibleShares.map((share) => (
            <li key={share.id}>
              <button
                type="button"
                onClick={() => setSelected(share)}
                className="flex w-full items-center gap-4 px-1 py-4 text-left transition-colors hover:bg-muted/40 focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
              >
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <p className="truncate font-medium">{shareTitle(share)}</p>
                    <ShareStatusBadge status={share.status} />
                  </div>
                  <p className="mt-1 truncate text-sm text-muted-foreground">
                    {resourceLabel(share, org.name, projectNames)} ·{" "}
                    {accessSummary(share.resourceType)}
                  </p>
                </div>
                <span className="hidden shrink-0 text-xs text-muted-foreground sm:block">
                  {expirySummary(share)}
                </span>
                <ChevronRightIcon
                  className="size-4 shrink-0 text-muted-foreground"
                  aria-hidden="true"
                />
              </button>
            </li>
          ))}
        </ul>
      ) : null}

      <ShareDetailsSheet
        share={selected}
        orgName={org.name}
        projectNames={projectNames}
        currentUserId={session.userId}
        canShare={canShare}
        onClose={() => setSelected(null)}
        onOpenResource={openResource}
        onReplace={startReplacement}
        onRevoke={(share) => setRevoking(share)}
      />

      {createTarget ? (
        <CreateShareDialog
          orgId={orgId}
          projects={projects.map((project) => ({
            id: project.id,
            name: project.name,
          }))}
          initialResourceType={createTarget.resourceType}
          initialProjectID={createTarget.projectID}
          onCreated={invalidate}
          onCancel={() => setCreateTarget(null)}
          onSessionInvalid={sessionInvalid}
        />
      ) : null}

      <AlertDialog
        open={revoking !== null}
        onOpenChange={(open) => {
          if (!open && !revokingBusy) {
            setRevoking(null)
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia>
              <BanIcon aria-hidden="true" />
            </AlertDialogMedia>
            <AlertDialogTitle>Revoke share link?</AlertDialogTitle>
            <AlertDialogDescription>
              Anyone using this link will immediately lose access. This cannot
              be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={revokingBusy}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={revokingBusy}
              onClick={() => void handleRevoke()}
            >
              {revokingBusy ? <Spinner data-icon="inline-start" /> : null}
              {revokingBusy ? "Revoking..." : "Revoke link"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

function ShareDetailsSheet({
  share,
  orgName,
  projectNames,
  currentUserId,
  canShare,
  onClose,
  onOpenResource,
  onReplace,
  onRevoke,
}: {
  share: ShareCredential | null
  orgName: string
  projectNames: Record<string, string>
  currentUserId: string
  canShare: boolean
  onClose: () => void
  onOpenResource: (share: ShareCredential) => void
  onReplace: (share: ShareCredential) => void
  onRevoke: (share: ShareCredential) => void
}) {
  return (
    <Sheet
      open={share !== null}
      onOpenChange={(open) => {
        if (!open) {
          onClose()
        }
      }}
    >
      <SheetContent side="right" className="flex flex-col sm:max-w-md">
        {share ? (
          <>
            <SheetHeader>
              <SheetTitle className="font-display text-xl">
                {shareTitle(share)}
              </SheetTitle>
              <SheetDescription>
                {resourceLabel(share, orgName, projectNames)} ·{" "}
                {accessSummary(share.resourceType)}
              </SheetDescription>
            </SheetHeader>

            <div className="flex min-h-0 flex-1 flex-col gap-5 overflow-y-auto px-4">
              <div className="flex flex-wrap items-center gap-2">
                <ShareStatusBadge status={share.status} />
                <span className="text-sm text-muted-foreground">
                  {expirySummary(share)}
                </span>
              </div>

              <dl className="grid gap-3 text-sm sm:grid-cols-2">
                <Meta label="Status" value={statusLabel(share.status)} />
                <Meta label="Expires" value={formatDate(share.expiresAt)} />
                <Meta label="Created" value={formatDate(share.createdAt)} />
                <Meta
                  label="Created by"
                  value={
                    share.createdBy === currentUserId ? "You" : share.createdBy
                  }
                />
              </dl>

              <div className="flex flex-col gap-3">
                <p className="text-label text-muted-foreground uppercase">
                  Permissions
                </p>
                {groupScopes(share.scopes).map((group) => (
                  <div key={group.label} className="flex flex-col gap-1.5">
                    <p className="text-sm font-medium">{group.label}</p>
                    <div className="flex flex-wrap gap-1.5">
                      {group.scopes.map((scope) => (
                        <Badge key={scope} variant="outline">
                          {scopeLabel(scope)}
                        </Badge>
                      ))}
                    </div>
                  </div>
                ))}
              </div>

              <p className="text-xs text-muted-foreground">
                The share URL is shown only when the link is created. Create a
                replacement if you need a copyable URL.
              </p>
            </div>

            <SheetFooter className="border-t border-border">
              <div className="flex flex-col gap-2">
                {share.resourceType !== "expense" ? (
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => onOpenResource(share)}
                  >
                    <ExternalLinkIcon data-icon="inline-start" />
                    Open{" "}
                    {share.resourceType === "org" ? "organization" : "project"}
                  </Button>
                ) : null}
                {canShare && share.resourceType !== "expense" ? (
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => onReplace(share)}
                  >
                    <RefreshCwIcon data-icon="inline-start" />
                    Create replacement
                  </Button>
                ) : null}
                {canShare && share.status === "active" ? (
                  <Button
                    type="button"
                    variant="destructive"
                    onClick={() => onRevoke(share)}
                  >
                    <BanIcon data-icon="inline-start" />
                    Revoke link
                  </Button>
                ) : null}
                <Button type="button" variant="ghost" onClick={onClose}>
                  Close
                </Button>
              </div>
            </SheetFooter>
          </>
        ) : null}
      </SheetContent>
    </Sheet>
  )
}

function shareTitle(share: ShareCredential): string {
  return share.name?.trim() || "Untitled share link"
}

function resourceLabel(
  share: ShareCredential,
  orgName: string,
  projectNames: Record<string, string>
): string {
  switch (share.resourceType) {
    case "org":
      return `Organization · ${orgName}`
    case "project":
      return `Project · ${projectNames[share.resourceId] ?? share.resourceId}`
    case "expense":
      return `Expense · ${share.resourceId}`
    default:
      return share.resourceType
  }
}

function ShareStatusBadge({ status }: { status: ShareCredential["status"] }) {
  return (
    <Badge
      variant={
        status === "active"
          ? "default"
          : status === "revoked"
            ? "destructive"
            : "secondary"
      }
    >
      {statusLabel(status)}
    </Badge>
  )
}

function statusLabel(status: ShareCredential["status"]): string {
  return status === "active"
    ? "Active"
    : status === "revoked"
      ? "Revoked"
      : "Expired"
}

function expirySummary(share: ShareCredential): string {
  if (share.status === "revoked") {
    return "Revoked"
  }
  if (share.status === "expired") {
    return "Expired"
  }
  if (!share.expiresAt) {
    return "No expiry"
  }
  const expires = new Date(share.expiresAt)
  if (Number.isNaN(expires.getTime())) {
    return share.expiresAt
  }
  const days = Math.ceil((expires.getTime() - Date.now()) / 86_400_000)
  if (days <= 0) {
    return "Expires today"
  }
  return days === 1 ? "Expires in 1 day" : `Expires in ${days} days`
}

function formatDate(value: string | undefined): string {
  if (!value) {
    return "—"
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
  }).format(date)
}

function Meta({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-label text-muted-foreground uppercase">{label}</dt>
      <dd className="mt-0.5 truncate text-foreground">{value}</dd>
    </div>
  )
}
