import { useCallback, useEffect, useId, useState } from "react"
import { useNavigate, useParams, useSearchParams } from "react-router"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { CircleAlertIcon, LinkIcon, PlusIcon } from "lucide-react"

import { PageHeader } from "@/components/page-header"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Spinner } from "@/components/ui/spinner"
import { queryKeys } from "@/app/query-keys"
import { useAuth, useSession } from "@/features/auth/auth-provider"
import { useOrgWorkspace } from "@/features/orgs/layouts/org-workspace-layout"
import { isSessionInvalidError, type Session } from "@/features/auth/session"
import { CreateLinkForm } from "@/features/links/components/create-link-form"
import { DeleteLinkDialog } from "@/features/links/components/delete-link-dialog"
import { EditLinkDialog } from "@/features/links/components/edit-link-dialog"
import { LinkFilterForm } from "@/features/links/components/link-filter-form"
import { LinkList } from "@/features/links/components/link-list"
import { QRCodeDialog } from "@/features/links/components/qr-code-dialog"
import {
  deleteLink,
  LINK_PAGE_SIZE,
  listLinks,
  updateLink,
} from "@/features/links/api"
import { linkErrorMessage } from "@/features/links/error-message"
import { useLinkStats } from "@/features/links/hooks/use-link-stats"
import type { LinkFilters, LinkPage, ShortLink } from "@/features/links/types"
import { useDebouncedValue } from "@/hooks/use-debounced-value"
import { usePageTitle } from "@/hooks/use-page-title"

type LoadStatus = "forbidden" | "loading" | "error" | "ready"

const EMPTY_PAGE: LinkPage = {
  items: [],
  nextOffset: null,
}

export function LinksPage() {
  const session = useSession()
  return <LinksView session={session} />
}

function LinksView({ session }: { session: Session }) {
  const { orgId = "" } = useParams<{ orgId: string }>()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const { logout } = useAuth()
  const queryClient = useQueryClient()
  const { org, roles, isOwner, canViewLinks } = useOrgWorkspace()
  const createDialogTriggerId = useId()

  const [query, setQuery] = useState("")
  const [tags, setTags] = useState<string[]>([])
  const [offset, setOffset] = useState(0)
  const [isCreateOpen, setIsCreateOpen] = useState(
    () => searchParams.get("new") === "1"
  )
  const [editing, setEditing] = useState<ShortLink | null>(null)
  const [qrLink, setQrLink] = useState<ShortLink | null>(null)
  const [deleting, setDeleting] = useState<ShortLink | null>(null)
  const [busyId, setBusyId] = useState<string | null>(null)
  const [actionError, setActionError] = useState("")

  const debouncedQuery = useDebouncedValue(query, 300)
  const canView = canViewLinks
  const canWrite =
    isOwner || roles.includes("link.editor") || roles.includes("link.admin")
  const canDelete = isOwner || roles.includes("link.admin")

  const filters: LinkFilters = { q: debouncedQuery, tags }

  const linksQuery = useQuery({
    queryKey: queryKeys.links(session.userId, orgId, filters, offset),
    queryFn: () => listLinks(orgId, filters, offset),
    enabled: canView,
  })

  const page = linksQuery.data ?? EMPTY_PAGE
  const status: LoadStatus = !canView
    ? "forbidden"
    : linksQuery.isPending
      ? "loading"
      : linksQuery.isError
        ? "error"
        : "ready"
  const error = linksQuery.isError ? linkErrorMessage(linksQuery.error) : ""
  const hasFilters = query.trim() !== "" || tags.length > 0
  const stats = useLinkStats(
    session.userId,
    orgId,
    page.items.map((link) => link.id),
    canView && status === "ready"
  )

  usePageTitle("Links · Pico")

  const sessionInvalid = useCallback(() => {
    void logout()
    navigate("/login", { replace: true })
  }, [logout, navigate])

  useEffect(() => {
    if (linksQuery.error && isSessionInvalidError(linksQuery.error)) {
      sessionInvalid()
    }
  }, [linksQuery.error, sessionInvalid])

  const invalidateLinks = useCallback(() => {
    void queryClient.invalidateQueries({
      queryKey: queryKeys.linksRoot(session.userId, orgId),
    })
  }, [orgId, queryClient, session.userId])

  function handleCreateOpenChange(open: boolean) {
    setIsCreateOpen(open)
    if (!open && searchParams.has("new")) {
      const next = new URLSearchParams(searchParams)
      next.delete("new")
      setSearchParams(next, { replace: true })
    }
  }

  function handleCreated() {
    handleCreateOpenChange(false)
    setOffset(0)
    invalidateLinks()
  }

  function handleUpdated() {
    setEditing(null)
    invalidateLinks()
  }

  async function handleToggleDisabled(link: ShortLink) {
    setBusyId(link.id)
    setActionError("")
    try {
      await updateLink(orgId, link.id, { disabled: !link.disabled })
      invalidateLinks()
    } catch (error) {
      if (isSessionInvalidError(error)) {
        sessionInvalid()
        return
      }
      setActionError(linkErrorMessage(error))
    } finally {
      setBusyId(null)
    }
  }

  async function handleDelete() {
    if (!deleting) {
      return
    }

    setBusyId(deleting.id)
    setActionError("")
    try {
      await deleteLink(orgId, deleting.id)
      const nextOffset =
        page.items.length === 1 && offset > 0
          ? Math.max(0, offset - LINK_PAGE_SIZE)
          : offset
      setDeleting(null)
      if (nextOffset !== offset) {
        setOffset(nextOffset)
      }
      invalidateLinks()
    } catch (error) {
      if (isSessionInvalidError(error)) {
        sessionInvalid()
        return
      }
      setDeleting(null)
      setActionError(linkErrorMessage(error))
    } finally {
      setBusyId(null)
    }
  }

  function clearFilters() {
    setQuery("")
    setTags([])
    setOffset(0)
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Short links"
        title="Links"
        description={`Short links for ${org.name}. Anyone with the link can open it; management stays private.`}
        actions={
          canWrite ? (
            <Dialog
              open={isCreateOpen}
              onOpenChange={handleCreateOpenChange}
              triggerId={createDialogTriggerId}
            >
              <DialogTrigger
                id={createDialogTriggerId}
                render={
                  <Button
                    type="button"
                    className="h-10 px-4 text-xs font-semibold tracking-[0.14em] uppercase"
                  />
                }
              >
                <PlusIcon data-icon="inline-start" />
                New link
              </DialogTrigger>
              <DialogContent className="max-h-[calc(100svh-2rem)] overflow-y-auto sm:max-w-lg">
                <DialogHeader>
                  <DialogTitle className="font-display text-2xl tracking-tight">
                    Create link
                  </DialogTitle>
                  <DialogDescription>
                    Pick a destination and an optional alias. Links are public
                    once created.
                  </DialogDescription>
                </DialogHeader>
                <CreateLinkForm
                  orgId={orgId}
                  onCreated={handleCreated}
                  onCancel={() => setIsCreateOpen(false)}
                  onSessionInvalid={sessionInvalid}
                />
              </DialogContent>
            </Dialog>
          ) : null
        }
      />

      {status === "forbidden" ? (
        <Card>
          <CardContent className="flex flex-col gap-4 py-8">
            <Alert>
              <CircleAlertIcon aria-hidden="true" />
              <AlertDescription>
                You do not have access to links in this organization.
              </AlertDescription>
            </Alert>
            <div>
              <Button
                type="button"
                variant="outline"
                onClick={() => navigate(`/orgs/${orgId}`)}
              >
                Back to organization
              </Button>
            </div>
          </CardContent>
        </Card>
      ) : (
        <>
          <LinkFilterForm
            query={query}
            tags={tags}
            onQueryChange={(value) => {
              setQuery(value)
              setOffset(0)
            }}
            onTagsChange={(nextTags) => {
              setTags(nextTags)
              setOffset(0)
            }}
            onClear={clearFilters}
          />

          {actionError ? (
            <Alert variant="destructive">
              <CircleAlertIcon aria-hidden="true" />
              <AlertDescription>{actionError}</AlertDescription>
            </Alert>
          ) : null}

          {status === "loading" ? (
            <Card>
              <CardContent className="flex items-center justify-center gap-2 py-12 text-sm text-muted-foreground">
                <Spinner />
                Loading links...
              </CardContent>
            </Card>
          ) : null}

          {status === "error" ? (
            <Card>
              <CardContent className="flex flex-col gap-4 py-6">
                <Alert variant="destructive">
                  <CircleAlertIcon aria-hidden="true" />
                  <AlertDescription>{error}</AlertDescription>
                </Alert>
                <div className="flex gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => void linksQuery.refetch()}
                  >
                    Try again
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    onClick={() => navigate(`/orgs/${orgId}`)}
                  >
                    Back to organization
                  </Button>
                </div>
              </CardContent>
            </Card>
          ) : null}

          {status === "ready" && page.items.length === 0 ? (
            <div className="flex flex-col items-center gap-3 border-y border-border py-14 text-center">
              <div className="flex size-12 items-center justify-center rounded-md border border-border bg-card text-muted-foreground">
                <LinkIcon className="size-5" aria-hidden="true" />
              </div>
              <p className="font-display text-2xl font-semibold tracking-tight">
                {hasFilters ? "No matching links" : "No links yet"}
              </p>
              <p className="max-w-sm text-sm text-muted-foreground">
                {hasFilters
                  ? "Try changing the alias search or tags."
                  : canWrite
                    ? "Use New link to create the first short link."
                    : "No links have been created in this organization yet."}
              </p>
            </div>
          ) : null}

          {status === "ready" && page.items.length > 0 ? (
            <>
              <LinkList
                links={page.items}
                clicksByLinkId={stats.clicksByLinkId}
                statsLoading={stats.isLoading}
                statsError={stats.isError}
                canWrite={canWrite}
                canDelete={canDelete}
                busyId={busyId}
                onEdit={setEditing}
                onShowQR={setQrLink}
                onToggleDisabled={(link) => void handleToggleDisabled(link)}
                onDelete={setDeleting}
              />

              <nav
                aria-label="Link pagination"
                className="flex items-center justify-end gap-2"
              >
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={offset === 0}
                  onClick={() =>
                    setOffset(Math.max(0, offset - LINK_PAGE_SIZE))
                  }
                >
                  Previous
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={page.nextOffset === null}
                  onClick={() => {
                    if (page.nextOffset !== null) {
                      setOffset(page.nextOffset)
                    }
                  }}
                >
                  Next
                </Button>
              </nav>
            </>
          ) : null}
        </>
      )}

      {editing ? (
        <EditLinkDialog
          orgId={orgId}
          link={editing}
          onUpdated={handleUpdated}
          onCancel={() => setEditing(null)}
          onSessionInvalid={sessionInvalid}
        />
      ) : null}

      {qrLink ? (
        <QRCodeDialog link={qrLink} onClose={() => setQrLink(null)} />
      ) : null}

      <DeleteLinkDialog
        link={deleting}
        deleting={deleting !== null && busyId === deleting.id}
        onCancel={() => setDeleting(null)}
        onConfirm={() => void handleDelete()}
      />
    </div>
  )
}
