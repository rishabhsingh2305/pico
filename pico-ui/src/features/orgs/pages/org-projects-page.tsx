import { useCallback, useEffect, useId, useState } from "react"
import { useNavigate, useParams } from "react-router"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { CircleAlertIcon, FolderIcon, PlusIcon, TagsIcon } from "lucide-react"

import { queryKeys } from "@/app/query-keys"
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
import { useAuth, useSession } from "@/features/auth/auth-provider"
import { isSessionInvalidError, type Session } from "@/features/auth/session"
import { useOrgWorkspace } from "@/features/orgs/layouts/org-workspace-layout"
import { orgErrorMessage } from "@/features/orgs/error-message"
import { CreateProjectForm } from "@/features/projects/components/create-project-form"
import { ProjectCard } from "@/features/projects/components/project-card"
import { ProjectFilterForm } from "@/features/projects/components/project-filter-form"
import { listProjects } from "@/features/projects/api"
import { getShareStatus } from "@/features/sharing/api"
import { usePageTitle } from "@/hooks/use-page-title"
import { ApiError } from "@/lib/api-client"

type LoadStatus = "loading" | "ready" | "error"

export function OrgProjectsPage() {
  const session = useSession()
  return <OrgProjectsView session={session} />
}

function OrgProjectsView({ session }: { session: Session }) {
  const { orgId = "" } = useParams<{ orgId: string }>()
  const navigate = useNavigate()
  const { logout } = useAuth()
  const queryClient = useQueryClient()
  const { org, canShare, canViewShares } = useOrgWorkspace()
  const projectDialogTriggerId = useId()

  const [projectTags, setProjectTags] = useState<string[]>([])
  const [isProjectDialogOpen, setIsProjectDialogOpen] = useState(false)

  const projectsQuery = useQuery({
    queryKey: queryKeys.projects(session.userId, orgId, projectTags),
    queryFn: () => listProjects(orgId, projectTags),
    enabled: orgId !== "",
  })

  const projects = projectsQuery.data ?? []
  const projectIDs = projects.map((project) => project.id)
  const shareStatusQuery = useQuery({
    queryKey: queryKeys.shareStatus(orgId, "project", projectIDs),
    queryFn: () => getShareStatus(orgId, "project", projectIDs),
    enabled: canViewShares && projectIDs.length > 0,
  })
  const sharedProjectIDs = new Set(
    (shareStatusQuery.data ?? []).map((item) => item.resourceId)
  )

  const status: LoadStatus = projectsQuery.isPending
    ? "loading"
    : projectsQuery.isError
      ? "error"
      : "ready"
  const error = projectsQuery.isError
    ? projectAccessErrorMessage(projectsQuery.error)
    : ""
  const showProjectFilter = projects.length > 0 || projectTags.length > 0

  usePageTitle(`${org.name} projects · Pico`)

  const sessionInvalid = useCallback(() => {
    void logout()
    navigate("/login", { replace: true })
  }, [logout, navigate])

  useEffect(() => {
    if (projectsQuery.error && isSessionInvalidError(projectsQuery.error)) {
      sessionInvalid()
    }
  }, [projectsQuery.error, sessionInvalid])

  useEffect(() => {
    if (!projectsQuery.data) {
      return
    }
    for (const project of projectsQuery.data) {
      queryClient.setQueryData(
        queryKeys.project(session.userId, orgId, project.id),
        project
      )
    }
  }, [orgId, projectsQuery.data, queryClient, session.userId])

  function changeProjectTags(nextTags: string[]) {
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
    void queryClient.invalidateQueries({
      queryKey: queryKeys.projectsRoot(session.userId, orgId),
    })
  }

  function handleProjectTagsUpdated() {
    void queryClient.invalidateQueries({
      queryKey: queryKeys.projectsRoot(session.userId, orgId),
    })
    void queryClient.invalidateQueries({
      queryKey: ["project", session.userId, orgId],
    })
  }

  function invalidateShareStatus() {
    void queryClient.invalidateQueries({
      queryKey: ["share-status", orgId],
    })
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Directory"
        title="Projects"
        description={`Projects in ${org.name}.`}
        actions={
          status === "ready" ? (
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
                    Projects own the currency used by their expenses and the
                    tags used to filter this list.
                  </DialogDescription>
                </DialogHeader>
                <CreateProjectForm
                  orgId={orgId}
                  onCreated={handleProjectCreated}
                  onCancel={() => setIsProjectDialogOpen(false)}
                  onSessionInvalid={sessionInvalid}
                />
              </DialogContent>
            </Dialog>
          ) : null
        }
      />

      {showProjectFilter ? (
        <ProjectFilterForm tags={projectTags} onChange={changeProjectTags} />
      ) : null}

      {status === "loading" ? (
        <Card>
          <CardContent className="flex items-center justify-center gap-2 py-10 text-sm text-muted-foreground">
            <Spinner />
            Loading projects...
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
            <Button
              type="button"
              variant="outline"
              className="w-fit"
              onClick={() => void projectsQuery.refetch()}
            >
              Try again
            </Button>
          </CardContent>
        </Card>
      ) : null}

      {status === "ready" && projects.length === 0 ? (
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

      {status === "ready" && projects.length > 0 ? (
        <ul className="grid gap-3 md:grid-cols-2">
          {projects.map((project) => (
            <li key={project.id}>
              <ProjectCard
                orgId={orgId}
                project={project}
                activeTags={projectTags}
                shared={sharedProjectIDs.has(project.id)}
                canShare={canShare}
                onToggleTag={toggleProjectTag}
                onTagsUpdated={handleProjectTagsUpdated}
                onShareChanged={invalidateShareStatus}
                onSessionInvalid={sessionInvalid}
              />
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  )
}

function projectAccessErrorMessage(error: unknown): string {
  if (error instanceof ApiError && error.status === 403) {
    return "You no longer have access to this organization."
  }
  if (error instanceof ApiError && error.status === 404) {
    return "Organization not found."
  }
  return orgErrorMessage(error)
}
