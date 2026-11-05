import { useCallback, useEffect, useId, useState } from "react"
import { Link, Navigate, useNavigate, useParams } from "react-router"
import {
  ArrowLeftIcon,
  CircleAlertIcon,
  FolderIcon,
  LayoutDashboardIcon,
  PlusIcon,
  TagsIcon,
} from "lucide-react"

import { AppShell } from "@/components/app-shell"
import { PageHeader } from "@/components/page-header"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
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
import {
  clearSession,
  getSession,
  isUserNotFoundError,
  type TemporarySession,
} from "@/features/auth/session"
import { getOrg } from "@/features/orgs/api"
import { orgErrorMessage } from "@/features/orgs/error-message"
import type { Org } from "@/features/orgs/types"
import { CreateProjectForm } from "@/features/projects/components/create-project-form"
import { ProjectCard } from "@/features/projects/components/project-card"
import { ProjectFilterForm } from "@/features/projects/components/project-filter-form"
import { listProjects } from "@/features/projects/api"
import type { Project } from "@/features/projects/types"
import { usePageTitle } from "@/hooks/use-page-title"
import { ApiError } from "@/lib/api-client"

type LoadStatus = "loading" | "ready" | "error"

export function OrgDetailPage() {
  const session = getSession()
  if (!session) {
    return <Navigate to="/signup" replace />
  }
  return <OrgDetailView session={session} />
}

function OrgDetailView({ session }: { session: TemporarySession }) {
  const { orgId = "" } = useParams<{ orgId: string }>()
  const navigate = useNavigate()
  const projectDialogTriggerId = useId()

  const [org, setOrg] = useState<Org | null>(null)
  const [status, setStatus] = useState<LoadStatus>("loading")
  const [error, setError] = useState("")
  const [attempt, setAttempt] = useState(0)

  const [projects, setProjects] = useState<Project[]>([])
  const [projectsStatus, setProjectsStatus] = useState<LoadStatus>("loading")
  const [projectsError, setProjectsError] = useState("")
  const [projectTags, setProjectTags] = useState<string[]>([])
  const [projectsAttempt, setProjectsAttempt] = useState(0)
  const [isProjectDialogOpen, setIsProjectDialogOpen] = useState(false)

  usePageTitle(org ? `${org.name} · Pico` : "Organization · Pico")

  const sessionInvalid = useCallback(() => {
    clearSession()
    navigate("/signup", { replace: true })
  }, [navigate])

  useEffect(() => {
    let cancelled = false

    getOrg(session.userId, orgId)
      .then((loadedOrg) => {
        if (!cancelled) {
          setOrg(loadedOrg)
          setStatus("ready")
        }
      })
      .catch((loadError: unknown) => {
        if (cancelled) {
          return
        }
        if (isUserNotFoundError(loadError)) {
          sessionInvalid()
          return
        }
        setError(orgAccessErrorMessage(loadError))
        setStatus("error")
      })

    return () => {
      cancelled = true
    }
  }, [orgId, session.userId, attempt, sessionInvalid])

  useEffect(() => {
    let cancelled = false

    listProjects(session.userId, orgId, projectTags)
      .then((loadedProjects) => {
        if (!cancelled) {
          setProjects(loadedProjects)
          setProjectsError("")
          setProjectsStatus("ready")
        }
      })
      .catch((loadError: unknown) => {
        if (cancelled) {
          return
        }
        if (isUserNotFoundError(loadError)) {
          sessionInvalid()
          return
        }
        setProjectsError(orgAccessErrorMessage(loadError))
        setProjectsStatus("error")
      })

    return () => {
      cancelled = true
    }
  }, [orgId, session.userId, projectTags, projectsAttempt, sessionInvalid])

  function retry() {
    setStatus("loading")
    setError("")
    setAttempt((current) => current + 1)
  }

  function retryProjects() {
    setProjectsStatus("loading")
    setProjectsError("")
    setProjectsAttempt((current) => current + 1)
  }

  function changeProjectTags(nextTags: string[]) {
    setProjectsStatus("loading")
    setProjectsError("")
    setProjectTags(nextTags)
  }

  function toggleProjectTag(tag: string) {
    const nextTags = projectTags.includes(tag)
      ? projectTags.filter((item) => item !== tag)
      : [...projectTags, tag]
    changeProjectTags(nextTags)
  }

  function handleProjectCreated() {
    setIsProjectDialogOpen(false)
    setProjectsStatus("loading")
    setProjectsAttempt((current) => current + 1)
  }

  function handleProjectTagsUpdated() {
    setProjectsStatus("loading")
    setProjectsAttempt((current) => current + 1)
  }

  const showProjectFilter = projects.length > 0 || projectTags.length > 0

  return (
    <AppShell session={session}>
      <div className="flex flex-col gap-8">
        <div>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="-ml-2 w-fit text-muted-foreground"
            onClick={() => navigate("/orgs")}
          >
            <ArrowLeftIcon data-icon="inline-start" />
            Organizations
          </Button>
        </div>

        {status === "loading" ? (
          <Card>
            <CardContent className="flex items-center justify-center gap-2 py-12 text-sm text-muted-foreground">
              <Spinner />
              Loading organization...
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
                <Button type="button" variant="outline" onClick={retry}>
                  Try again
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => navigate("/orgs")}
                >
                  Back to organizations
                </Button>
              </div>
            </CardContent>
          </Card>
        ) : null}

        {status === "ready" && org ? (
          <>
            <PageHeader
              eyebrow="Organization"
              title={org.name}
              description="Choose a project to view and manage its expenses."
              actions={
                <Badge
                  variant={
                    org.userId === session.userId ? "default" : "secondary"
                  }
                >
                  {org.userId === session.userId ? "Owner" : "Member"}
                </Badge>
              }
            />

            {org.tags.length > 0 ? (
              <div className="flex flex-wrap gap-2">
                {org.tags.map((tag) => (
                  <span
                    key={tag}
                    className="inline-flex items-center rounded-sm bg-secondary px-2 py-1 text-[10px] font-semibold tracking-[0.14em] text-secondary-foreground uppercase"
                  >
                    {tag}
                  </span>
                ))}
              </div>
            ) : null}

            <section className="flex flex-col gap-5">
              <div className="flex flex-wrap items-end justify-between gap-4">
                <div>
                  <p className="text-label text-muted-foreground uppercase">
                    Directory
                  </p>
                  <div className="mt-1 flex items-center gap-3">
                    <h2 className="font-display text-heading">Projects</h2>
                    {projectsStatus === "ready" && projects.length > 0 ? (
                      <Badge variant="outline">{projects.length}</Badge>
                    ) : null}
                  </div>
                </div>
                <div className="flex flex-wrap items-center gap-2">
                  <Link
                    to={`/orgs/${orgId}/dashboard`}
                    className={buttonVariants({
                      variant: "outline",
                      className:
                        "h-10 px-4 text-xs font-semibold tracking-[0.14em] uppercase",
                    })}
                  >
                    <LayoutDashboardIcon data-icon="inline-start" />
                    Dashboard
                  </Link>
                  {projectsStatus === "ready" ? (
                    <Dialog
                      open={isProjectDialogOpen}
                      onOpenChange={setIsProjectDialogOpen}
                      triggerId={projectDialogTriggerId}
                    >
                      <DialogTrigger
                        id={projectDialogTriggerId}
                        render={
                          <Button
                            type="button"
                            className="h-10 px-4 text-xs font-semibold tracking-[0.14em] uppercase"
                          />
                        }
                      >
                        <PlusIcon data-icon="inline-start" />
                        New project
                      </DialogTrigger>
                      <DialogContent className="max-h-[calc(100svh-2rem)] overflow-y-auto sm:max-w-lg">
                        <DialogHeader>
                          <DialogTitle className="font-display text-2xl tracking-tight">
                            Create project
                          </DialogTitle>
                          <DialogDescription>
                            Projects own the currency used by their expenses and
                            the tags used to filter this list.
                          </DialogDescription>
                        </DialogHeader>
                        <CreateProjectForm
                          userId={session.userId}
                          orgId={orgId}
                          onCreated={() => handleProjectCreated()}
                          onCancel={() => setIsProjectDialogOpen(false)}
                          onSessionInvalid={sessionInvalid}
                        />
                      </DialogContent>
                    </Dialog>
                  ) : null}
                </div>
              </div>

              {showProjectFilter ? (
                <ProjectFilterForm
                  tags={projectTags}
                  onChange={changeProjectTags}
                />
              ) : null}

              {projectsStatus === "loading" ? (
                <Card>
                  <CardContent className="flex items-center justify-center gap-2 py-10 text-sm text-muted-foreground">
                    <Spinner />
                    Loading projects...
                  </CardContent>
                </Card>
              ) : null}

              {projectsStatus === "error" ? (
                <Card>
                  <CardContent className="flex flex-col gap-4 py-6">
                    <Alert variant="destructive">
                      <CircleAlertIcon aria-hidden="true" />
                      <AlertDescription>{projectsError}</AlertDescription>
                    </Alert>
                    <Button
                      type="button"
                      variant="outline"
                      className="w-fit"
                      onClick={retryProjects}
                    >
                      Try again
                    </Button>
                  </CardContent>
                </Card>
              ) : null}

              {projectsStatus === "ready" && projects.length === 0 ? (
                <Card>
                  <CardContent className="flex flex-col items-center gap-2 py-10 text-center">
                    <div className="flex size-11 items-center justify-center rounded-full bg-muted text-muted-foreground">
                      {projectTags.length > 0 ? (
                        <TagsIcon className="size-5" aria-hidden="true" />
                      ) : (
                        <FolderIcon className="size-5" aria-hidden="true" />
                      )}
                    </div>
                    <p className="font-medium">
                      {projectTags.length > 0
                        ? "No projects match these tags"
                        : "No projects yet"}
                    </p>
                    <p className="text-sm text-muted-foreground">
                      {projectTags.length > 0
                        ? "Remove a tag or clear the filter."
                        : "Use New project to start adding expenses."}
                    </p>
                  </CardContent>
                </Card>
              ) : null}

              {projectsStatus === "ready" && projects.length > 0 ? (
                <ul className="grid gap-3 md:grid-cols-2">
                  {projects.map((project) => (
                    <li key={project.id}>
                      <ProjectCard
                        userId={session.userId}
                        orgId={orgId}
                        project={project}
                        activeTags={projectTags}
                        onToggleTag={toggleProjectTag}
                        onTagsUpdated={handleProjectTagsUpdated}
                        onSessionInvalid={sessionInvalid}
                      />
                    </li>
                  ))}
                </ul>
              ) : null}
            </section>
          </>
        ) : null}
      </div>
    </AppShell>
  )
}

function orgAccessErrorMessage(error: unknown): string {
  if (error instanceof ApiError && error.status === 403) {
    return "You no longer have access to this organization."
  }
  if (error instanceof ApiError && error.status === 404) {
    return "Organization not found."
  }
  return orgErrorMessage(error)
}
