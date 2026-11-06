import { useCallback, useEffect, useState } from "react"
import { useNavigate } from "react-router"
import { Building2Icon, CircleAlertIcon, TagsIcon } from "lucide-react"

import { AppShell } from "@/components/app-shell"
import { PageHeader } from "@/components/page-header"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Spinner } from "@/components/ui/spinner"
import { useAuth, useSession } from "@/features/auth/auth-provider"
import { isSessionInvalidError, type Session } from "@/features/auth/session"
import { CreateOrgForm } from "@/features/orgs/components/create-org-form"
import { OrgCard } from "@/features/orgs/components/org-card"
import { OrgFilterForm } from "@/features/orgs/components/org-filter-form"
import { listOrgs } from "@/features/orgs/api"
import { orgErrorMessage } from "@/features/orgs/error-message"
import type { Org } from "@/features/orgs/types"
import { usePageTitle } from "@/hooks/use-page-title"

type LoadStatus = "loading" | "ready" | "error"

export function OrgsPage() {
  const session = useSession()
  return <OrgsView session={session} />
}

function OrgsView({ session }: { session: Session }) {
  usePageTitle("Organizations · Pico")
  const navigate = useNavigate()
  const { logout } = useAuth()

  const [orgs, setOrgs] = useState<Org[]>([])
  const [tags, setTags] = useState<string[]>([])
  const [status, setStatus] = useState<LoadStatus>("loading")
  const [loadError, setLoadError] = useState("")
  const [attempt, setAttempt] = useState(0)

  const handleSessionInvalid = useCallback(() => {
    void logout()
    navigate("/login", { replace: true })
  }, [logout, navigate])

  useEffect(() => {
    let cancelled = false

    listOrgs(session.userId, tags)
      .then((items) => {
        if (!cancelled) {
          setOrgs(items)
          setStatus("ready")
        }
      })
      .catch((error: unknown) => {
        if (cancelled) {
          return
        }
        if (isSessionInvalidError(error)) {
          handleSessionInvalid()
          return
        }
        setLoadError(orgErrorMessage(error))
        setStatus("error")
      })

    return () => {
      cancelled = true
    }
  }, [session.userId, tags, attempt, navigate, handleSessionInvalid])

  function retry() {
    setStatus("loading")
    setLoadError("")
    setAttempt((current) => current + 1)
  }

  function handleCreated(org: Org) {
    if (tags.length === 0) {
      setOrgs((current) => [
        org,
        ...current.filter((item) => item.id !== org.id),
      ])
      setStatus("ready")
      return
    }
    setStatus("loading")
    setLoadError("")
    setAttempt((current) => current + 1)
  }

  function changeTags(nextTags: string[]) {
    setStatus("loading")
    setLoadError("")
    setTags(nextTags)
  }

  function toggleTag(tag: string) {
    const nextTags = tags.includes(tag)
      ? tags.filter((item) => item !== tag)
      : [...tags, tag]
    changeTags(nextTags)
  }

  function handleTagsUpdated() {
    setStatus("loading")
    setLoadError("")
    setAttempt((current) => current + 1)
  }

  return (
    <AppShell session={session}>
      <div className="flex flex-col gap-8">
        <PageHeader
          eyebrow="Workspace"
          title="Organizations"
          description="Create an organization to group projects and track shared expenses."
        />

        <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_22rem] lg:items-start">
          <section
            aria-labelledby="organizations-heading"
            className="order-2 flex flex-col gap-4 lg:order-1"
          >
            <div className="flex items-end justify-between gap-3">
              <div>
                <p className="text-label text-muted-foreground uppercase">
                  Directory
                </p>
                <div className="mt-1 flex items-center gap-3">
                  <h2
                    id="organizations-heading"
                    className="font-display text-heading"
                  >
                    Your organizations
                  </h2>
                  {status === "ready" && orgs.length > 0 ? (
                    <Badge variant="outline">{orgs.length}</Badge>
                  ) : null}
                </div>
              </div>
            </div>

            {orgs.length > 0 || tags.length > 0 ? (
              <OrgFilterForm tags={tags} onChange={changeTags} />
            ) : null}

            {status === "loading" ? (
              <Card>
                <CardContent className="flex items-center justify-center gap-2 py-12 text-sm text-muted-foreground">
                  <Spinner />
                  Loading organizations...
                </CardContent>
              </Card>
            ) : null}

            {status === "error" ? (
              <Card>
                <CardContent className="flex flex-col gap-3 py-6">
                  <Alert variant="destructive">
                    <CircleAlertIcon aria-hidden="true" />
                    <AlertDescription>{loadError}</AlertDescription>
                  </Alert>
                  <div>
                    <Button type="button" variant="outline" onClick={retry}>
                      Try again
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ) : null}

            {status === "ready" && orgs.length === 0 ? (
              <Card>
                <CardContent className="flex flex-col items-center gap-3 py-12 text-center">
                  <div className="flex size-12 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                    {tags.length > 0 ? (
                      <TagsIcon className="size-5" aria-hidden="true" />
                    ) : (
                      <Building2Icon className="size-5" aria-hidden="true" />
                    )}
                  </div>
                  <div>
                    <p className="font-medium">
                      {tags.length > 0
                        ? "No organizations match these tags"
                        : "No organizations yet"}
                    </p>
                    <p className="mt-1 text-sm text-muted-foreground">
                      {tags.length > 0
                        ? "Remove a tag or clear the filter."
                        : "Create your first organization to get started."}
                    </p>
                  </div>
                </CardContent>
              </Card>
            ) : null}

            {status === "ready" && orgs.length > 0 ? (
              <ul className="flex flex-col gap-3">
                {orgs.map((org) => (
                  <li key={org.id}>
                    <OrgCard
                      userId={session.userId}
                      org={org}
                      activeTags={tags}
                      onToggleTag={toggleTag}
                      onTagsUpdated={handleTagsUpdated}
                      onSessionInvalid={handleSessionInvalid}
                    />
                  </li>
                ))}
              </ul>
            ) : null}
          </section>

          <aside className="order-1 lg:sticky lg:top-6 lg:order-2">
            <CreateOrgForm
              userId={session.userId}
              onCreated={handleCreated}
              onSessionInvalid={handleSessionInvalid}
            />
          </aside>
        </div>
      </div>
    </AppShell>
  )
}
