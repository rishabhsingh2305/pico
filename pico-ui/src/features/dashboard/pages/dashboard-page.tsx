import { useCallback, useEffect, useState, type ReactNode } from "react"
import { Link, useNavigate, useParams } from "react-router"
import { ArrowLeftIcon, CircleAlertIcon } from "lucide-react"

import { AppShell } from "@/components/app-shell"
import { PageHeader } from "@/components/page-header"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button, buttonVariants } from "@/components/ui/button"
import { Spinner } from "@/components/ui/spinner"
import { useAuth, useSession } from "@/features/auth/auth-provider"
import { isSessionInvalidError, type Session } from "@/features/auth/session"
import { DashboardSummary } from "@/features/dashboard/components/dashboard-summary"
import { ProjectFlow } from "@/features/dashboard/components/project-flow"
import { RecentActivity } from "@/features/dashboard/components/recent-activity"
import {
  summarizeCurrencies,
  type ProjectTotals,
} from "@/features/dashboard/data"
import { listExpenses } from "@/features/expenses/api"
import { expenseErrorMessage } from "@/features/expenses/error-message"
import type { Expense } from "@/features/expenses/types"
import { getOrg } from "@/features/orgs/api"
import type { Org } from "@/features/orgs/types"
import { listProjects } from "@/features/projects/api"
import { usePageTitle } from "@/hooks/use-page-title"
import { ApiError } from "@/lib/api-client"
import { cn } from "cn"

type LoadStatus = "loading" | "ready" | "error"

const RECENT_LIMIT = 6

type DashboardData = {
  projects: ProjectTotals[]
  recent: Expense[]
}

export function OrgDashboardPage() {
  const session = useSession()
  return <OrgDashboardView session={session} />
}

function OrgDashboardView({ session }: { session: Session }) {
  const { orgId = "" } = useParams<{ orgId: string }>()
  const navigate = useNavigate()
  const { logout } = useAuth()

  const [org, setOrg] = useState<Org | null>(null)
  const [data, setData] = useState<DashboardData | null>(null)
  const [status, setStatus] = useState<LoadStatus>("loading")
  const [error, setError] = useState("")
  const [attempt, setAttempt] = useState(0)
  const [selectedCurrency, setSelectedCurrency] = useState("")

  usePageTitle(org ? `${org.name} dashboard · Pico` : "Dashboard · Pico")

  const sessionInvalid = useCallback(() => {
    void logout()
    navigate("/login", { replace: true })
  }, [logout, navigate])

  useEffect(() => {
    let cancelled = false

    async function load() {
      try {
        const [loadedOrg, projects, recentPage] = await Promise.all([
          getOrg(session.userId, orgId),
          listProjects(session.userId, orgId),
          listExpenses({
            userId: session.userId,
            orgId,
            tags: [],
            limit: RECENT_LIMIT,
            offset: 0,
          }),
        ])

        const projectTotals = await Promise.all(
          projects.map(async (project) => ({
            project,
            totals: await listExpenses({
              userId: session.userId,
              orgId,
              projectId: project.id,
              tags: [],
              limit: 1,
              offset: 0,
            }),
          }))
        )

        if (!cancelled) {
          setOrg(loadedOrg)
          setData({ projects: projectTotals, recent: recentPage.items })
          setStatus("ready")
        }
      } catch (loadError: unknown) {
        if (cancelled) {
          return
        }
        if (isSessionInvalidError(loadError)) {
          sessionInvalid()
          return
        }
        setError(dashboardErrorMessage(loadError))
        setStatus("error")
      }
    }

    void load()

    return () => {
      cancelled = true
    }
  }, [orgId, session.userId, attempt, sessionInvalid])

  function retry() {
    setStatus("loading")
    setError("")
    setAttempt((current) => current + 1)
  }

  const summaries = data ? summarizeCurrencies(data.projects) : []
  const activeCurrency =
    summaries.find((summary) => summary.currency === selectedCurrency) ??
    summaries[0] ??
    null

  return (
    <AppShell session={session}>
      <div className="flex flex-col gap-8">
        <div>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="-ml-2 w-fit text-label text-muted-foreground uppercase"
            onClick={() => navigate(`/orgs/${orgId}`)}
          >
            <ArrowLeftIcon data-icon="inline-start" />
            Projects
          </Button>
        </div>

        <PageHeader
          eyebrow="Organization"
          title="Dashboard"
          description={
            org
              ? `Net flow and recent activity for ${org.name}.`
              : "Net flow and recent activity across this organization's projects."
          }
        />

        {status === "loading" ? (
          <LoadingPanel label="Loading dashboard..." />
        ) : null}

        {status === "error" ? (
          <ErrorPanel message={error} onRetry={retry} />
        ) : null}

        {status === "ready" && org && data ? (
          data.projects.length === 0 ? (
            <EmptyState
              eyebrow="No projects"
              title="Nothing to summarize"
              description={`${org.name} has no projects yet. Add a project to start recording debits and credits.`}
              action={
                <Link to={`/orgs/${org.id}`} className={cn(buttonVariants())}>
                  Create a project
                </Link>
              }
            />
          ) : activeCurrency ? (
            <div className="flex flex-col gap-12">
              <DashboardSummary
                summary={activeCurrency}
                currencies={summaries.map((summary) => summary.currency)}
                selectedCurrency={activeCurrency.currency}
                onCurrencyChange={setSelectedCurrency}
              />

              <div className="grid gap-12 lg:grid-cols-[minmax(0,1.3fr)_minmax(21rem,1fr)] lg:items-start">
                <ProjectFlow summary={activeCurrency} orgId={org.id} />
                <RecentActivity
                  expenses={data.recent}
                  projects={data.projects}
                  orgId={org.id}
                />
              </div>
            </div>
          ) : null
        ) : null}
      </div>
    </AppShell>
  )
}

function dashboardErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status === 403) {
      return "You no longer have access to this organization."
    }
    if (
      error.status === 404 &&
      (error.message === "org not found" ||
        error.message === "organization not found")
    ) {
      return "Organization not found."
    }
  }
  return expenseErrorMessage(error)
}

function LoadingPanel({ label }: { label: string }) {
  return (
    <div className="flex items-center justify-center gap-2 border-y border-border py-16 text-sm text-muted-foreground">
      <Spinner />
      {label}
    </div>
  )
}

function ErrorPanel({
  message,
  onRetry,
}: {
  message: string
  onRetry: () => void
}) {
  return (
    <div className="flex flex-col gap-4 border-y border-border py-8">
      <Alert variant="destructive">
        <CircleAlertIcon aria-hidden="true" />
        <AlertDescription>{message}</AlertDescription>
      </Alert>
      <Button
        type="button"
        variant="outline"
        className="w-fit"
        onClick={onRetry}
      >
        Try again
      </Button>
    </div>
  )
}

function EmptyState({
  eyebrow,
  title,
  description,
  action,
}: {
  eyebrow: string
  title: string
  description: string
  action: ReactNode
}) {
  return (
    <section className="flex flex-col items-start gap-4 border-y border-border py-16">
      <div className="flex items-center gap-3">
        <span className="h-px w-8 bg-primary" aria-hidden="true" />
        <p className="text-label text-primary uppercase">{eyebrow}</p>
      </div>
      <h2 className="font-display text-heading">{title}</h2>
      <p className="max-w-xl text-sm leading-6 text-muted-foreground">
        {description}
      </p>
      {action}
    </section>
  )
}
