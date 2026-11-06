import { useCallback, useEffect, useId, useState } from "react"
import { useNavigate, useParams } from "react-router"
import { ArrowLeftIcon, CircleAlertIcon, PlusIcon } from "lucide-react"

import { AppShell } from "@/components/app-shell"
import { CountUp } from "@/components/count-up"
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
import { CreateExpenseForm } from "@/features/expenses/components/create-expense-form"
import { ExpenseFilterForm } from "@/features/expenses/components/expense-filter-form"
import { ExpenseList } from "@/features/expenses/components/expense-list"
import { deleteExpense, listExpenses } from "@/features/expenses/api"
import { expenseErrorMessage } from "@/features/expenses/error-message"
import type { ExpenseFilters } from "@/features/expenses/filters"
import { formatMoney } from "@/features/expenses/money"
import type { Expense, ExpensePage } from "@/features/expenses/types"
import { getOrg } from "@/features/orgs/api"
import { orgErrorMessage } from "@/features/orgs/error-message"
import type { Org } from "@/features/orgs/types"
import { getProject } from "@/features/projects/api"
import type { Project } from "@/features/projects/types"
import { usePageTitle } from "@/hooks/use-page-title"
import { ApiError } from "@/lib/api-client"
import { cn } from "cn"

type LoadStatus = "loading" | "ready" | "error"

const PAGE_SIZE = 20
const EMPTY_PAGE: ExpensePage = {
  items: [],
  totalCount: 0,
  totalDebit: 0,
  totalCredit: 0,
  netTotal: 0,
}

export function ProjectDetailPage() {
  const session = useSession()
  return <ProjectDetailView session={session} />
}

function ProjectDetailView({ session }: { session: Session }) {
  const { orgId = "", projectId = "" } = useParams<{
    orgId: string
    projectId: string
  }>()
  const navigate = useNavigate()
  const { logout } = useAuth()
  const expenseDialogTriggerId = useId()

  const [org, setOrg] = useState<Org | null>(null)
  const [project, setProject] = useState<Project | null>(null)
  const [status, setStatus] = useState<LoadStatus>("loading")
  const [error, setError] = useState("")
  const [attempt, setAttempt] = useState(0)

  const [expensePage, setExpensePage] = useState<ExpensePage>(EMPTY_PAGE)
  const [expensesStatus, setExpensesStatus] = useState<LoadStatus>("loading")
  const [expensesError, setExpensesError] = useState("")
  const [deleteError, setDeleteError] = useState("")
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [filters, setFilters] = useState<ExpenseFilters>({ tags: [] })
  const [offset, setOffset] = useState(0)
  const [refreshKey, setRefreshKey] = useState(0)
  const [isExpenseDialogOpen, setIsExpenseDialogOpen] = useState(false)

  usePageTitle(project ? `${project.name} · Pico` : "Project · Pico")

  const sessionInvalid = useCallback(() => {
    void logout()
    navigate("/login", { replace: true })
  }, [logout, navigate])

  useEffect(() => {
    let cancelled = false

    Promise.all([
      getOrg(session.userId, orgId),
      getProject(session.userId, orgId, projectId),
    ])
      .then(([loadedOrg, loadedProject]) => {
        if (!cancelled) {
          setOrg(loadedOrg)
          setProject(loadedProject)
          setStatus("ready")
        }
      })
      .catch((loadError: unknown) => {
        if (cancelled) {
          return
        }
        if (isSessionInvalidError(loadError)) {
          sessionInvalid()
          return
        }
        setError(projectErrorMessage(loadError))
        setStatus("error")
      })

    return () => {
      cancelled = true
    }
  }, [orgId, projectId, session.userId, attempt, sessionInvalid])

  useEffect(() => {
    let cancelled = false

    listExpenses({
      userId: session.userId,
      orgId,
      projectId,
      tags: filters.tags,
      from: filters.from,
      to: filters.to,
      limit: PAGE_SIZE,
      offset,
    })
      .then((loadedPage) => {
        if (!cancelled) {
          setExpensePage(loadedPage)
          setExpensesError("")
          setExpensesStatus("ready")
        }
      })
      .catch((loadError: unknown) => {
        if (cancelled) {
          return
        }
        if (isSessionInvalidError(loadError)) {
          sessionInvalid()
          return
        }
        setExpensesError(projectErrorMessage(loadError))
        setExpensesStatus("error")
      })

    return () => {
      cancelled = true
    }
  }, [
    orgId,
    projectId,
    session.userId,
    filters,
    offset,
    refreshKey,
    sessionInvalid,
  ])

  function retryProject() {
    setStatus("loading")
    setError("")
    setAttempt((current) => current + 1)
  }

  function retryExpenses() {
    setExpensesStatus("loading")
    setExpensesError("")
    setRefreshKey((current) => current + 1)
  }

  function handleFiltersApplied(nextFilters: ExpenseFilters) {
    setExpensesStatus("loading")
    setExpensesError("")
    setDeleteError("")
    setFilters(nextFilters)
    setOffset(0)
  }

  function handleFiltersCleared() {
    setExpensesStatus("loading")
    setExpensesError("")
    setDeleteError("")
    setFilters({ tags: [] })
    setOffset(0)
  }

  function handleExpenseChanged() {
    setExpensesStatus("loading")
    setExpensesError("")
    setDeleteError("")
    setRefreshKey((current) => current + 1)
  }

  function changePage(nextOffset: number) {
    if (nextOffset === offset) {
      return
    }
    setExpensesStatus("loading")
    setExpensesError("")
    setDeleteError("")
    setOffset(nextOffset)
  }

  async function handleDelete(expense: Expense) {
    setDeletingId(expense.id)
    setDeleteError("")
    try {
      await deleteExpense(session.userId, orgId, expense.id)
      const remainingCount = expensePage.totalCount - 1
      setExpensesStatus("loading")
      if (offset > 0 && remainingCount <= offset) {
        setOffset(Math.max(0, offset - PAGE_SIZE))
      } else {
        setRefreshKey((current) => current + 1)
      }
    } catch (deleteFailure) {
      if (isSessionInvalidError(deleteFailure)) {
        sessionInvalid()
        return
      }
      setDeleteError(expenseErrorMessage(deleteFailure))
    } finally {
      setDeletingId(null)
    }
  }

  const hasFilters =
    filters.tags.length > 0 ||
    filters.from !== undefined ||
    filters.to !== undefined
  const currency = project?.currency ?? "USD"
  const summary = `${expensePage.totalCount} ${
    expensePage.totalCount === 1 ? "expense" : "expenses"
  } · ${formatMoney(expensePage.netTotal, currency)} net · ${formatMoney(
    expensePage.totalCredit,
    currency
  )} credit · ${formatMoney(expensePage.totalDebit, currency)} debit`
  const showingStart = expensePage.totalCount === 0 ? 0 : offset + 1
  const showingEnd = offset + expensePage.items.length

  return (
    <AppShell session={session}>
      <div className="flex flex-col gap-6 lg:h-[calc(100svh-7rem)] lg:min-h-[36rem] lg:overflow-x-hidden lg:overflow-y-auto">
        {status !== "ready" ? (
          <div>
            <BackToProjects onClick={() => navigate(`/orgs/${orgId}`)} />
          </div>
        ) : null}

        {status === "loading" ? (
          <Card>
            <CardContent className="flex items-center justify-center gap-2 py-12 text-sm text-muted-foreground">
              <Spinner />
              Loading project...
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
                <Button type="button" variant="outline" onClick={retryProject}>
                  Try again
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => navigate(`/orgs/${orgId}`)}
                >
                  Back to projects
                </Button>
              </div>
            </CardContent>
          </Card>
        ) : null}

        {status === "ready" && org && project ? (
          <div className="flex min-h-0 flex-1 flex-col gap-5">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div className="flex min-w-0 flex-wrap items-center gap-x-4 gap-y-2">
                <BackToProjects onClick={() => navigate(`/orgs/${orgId}`)} />
                <div className="flex items-center gap-2">
                  <span className="h-px w-6 bg-primary" aria-hidden="true" />
                  <p className="text-label text-primary uppercase">Project</p>
                </div>
                <h1 className="font-display text-heading text-balance">
                  {project.name}
                </h1>
                <p className="text-sm text-muted-foreground">
                  Expenses in {org.name}
                </p>
                {project.tags.length > 0 ? (
                  <div className="flex flex-wrap gap-1.5">
                    {project.tags.map((tag) => (
                      <span
                        key={tag}
                        className="inline-flex items-center rounded-sm bg-secondary px-2 py-0.5 text-[10px] font-semibold tracking-[0.14em] text-secondary-foreground uppercase"
                      >
                        {tag}
                      </span>
                    ))}
                  </div>
                ) : null}
              </div>

              <Dialog
                open={isExpenseDialogOpen}
                onOpenChange={setIsExpenseDialogOpen}
                triggerId={expenseDialogTriggerId}
              >
                <DialogTrigger
                  id={expenseDialogTriggerId}
                  render={
                    <Button
                      type="button"
                      className="h-10 w-fit px-4 text-xs font-semibold tracking-[0.14em] uppercase"
                    />
                  }
                >
                  <PlusIcon data-icon="inline-start" />
                  New expense
                </DialogTrigger>
                <DialogContent className="max-h-[calc(100svh-2rem)] overflow-y-auto sm:max-w-md">
                  <DialogHeader>
                    <DialogTitle className="font-display text-2xl tracking-tight">
                      Add expense
                    </DialogTitle>
                    <DialogDescription>
                      Record a debit or credit for this project in{" "}
                      {project.currency}.
                    </DialogDescription>
                  </DialogHeader>
                  <CreateExpenseForm
                    userId={session.userId}
                    orgId={orgId}
                    projectId={projectId}
                    currency={project.currency}
                    onCreated={() => {
                      setIsExpenseDialogOpen(false)
                      handleExpenseChanged()
                    }}
                    onCancel={() => setIsExpenseDialogOpen(false)}
                    onSessionInvalid={sessionInvalid}
                  />
                </DialogContent>
              </Dialog>
            </div>

            <div className="@container">
              <div className="relative overflow-hidden rounded-lg border border-border bg-card">
                <span
                  className={cn(
                    "absolute inset-y-0 left-0 w-1.5 lg:hidden",
                    expensePage.netTotal < 0 ? "bg-primary" : "bg-chart-2"
                  )}
                  aria-hidden="true"
                />
                <span
                  className={cn(
                    "absolute inset-x-0 top-0 hidden h-1.5 lg:block",
                    expensePage.netTotal < 0 ? "bg-primary" : "bg-chart-2"
                  )}
                  aria-hidden="true"
                />
                <div className="p-5 sm:p-6">
                  <div className="flex flex-wrap items-start justify-between gap-x-6 gap-y-2">
                    <span className="inline-flex bg-foreground px-2 py-1 font-mono text-[10px] font-semibold tracking-[0.16em] text-background uppercase">
                      {hasFilters ? "Filtered net total" : "Net total"}
                    </span>
                    <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
                      <p
                        className="font-mono text-[11px] tracking-[0.06em] text-muted-foreground uppercase sm:text-xs"
                        aria-hidden="true"
                      >
                        <span className="text-chart-2">
                          {formatMoney(expensePage.totalCredit, currency)}{" "}
                          credit
                        </span>
                        <span className="px-1.5">−</span>
                        <span className="text-primary">
                          {formatMoney(expensePage.totalDebit, currency)} debit
                        </span>
                      </p>
                      <span className="font-mono text-[10px] tracking-[0.16em] text-muted-foreground uppercase">
                        {currency}
                      </span>
                      {expensesStatus === "loading" ? (
                        <span className="flex items-center gap-1.5 font-mono text-[10px] tracking-[0.12em] text-muted-foreground uppercase">
                          <Spinner className="size-3" />
                          Updating
                        </span>
                      ) : null}
                    </div>
                  </div>

                  <span
                    aria-hidden="true"
                    className={cn(
                      expensePage.netTotal < 0 && "text-destructive"
                    )}
                  >
                    <CountUp
                      value={expensePage.netTotal}
                      format={(amount) => formatMoney(amount, currency)}
                      className="mt-3 block max-w-full font-display text-metric-fluid whitespace-nowrap tabular-nums"
                      durationMs={800}
                    />
                  </span>

                  <p className="sr-only" aria-live="polite">
                    {summary}
                  </p>
                </div>
              </div>
            </div>

            <ExpenseFilterForm
              onApply={handleFiltersApplied}
              onClear={handleFiltersCleared}
            />

            <section
              aria-label="Expenses"
              className="flex flex-col gap-4 lg:min-h-0 lg:flex-1"
            >
              {deleteError ? (
                <Alert variant="destructive">
                  <CircleAlertIcon aria-hidden="true" />
                  <AlertDescription>{deleteError}</AlertDescription>
                </Alert>
              ) : null}

              {expensesStatus === "loading" ? (
                <Card>
                  <CardContent className="flex items-center justify-center gap-2 py-10 text-sm text-muted-foreground">
                    <Spinner />
                    Loading expenses...
                  </CardContent>
                </Card>
              ) : null}

              {expensesStatus === "error" ? (
                <Card>
                  <CardContent className="flex flex-col gap-4 py-6">
                    <Alert variant="destructive">
                      <CircleAlertIcon aria-hidden="true" />
                      <AlertDescription>{expensesError}</AlertDescription>
                    </Alert>
                    <Button
                      type="button"
                      variant="outline"
                      className="w-fit"
                      onClick={retryExpenses}
                    >
                      Try again
                    </Button>
                  </CardContent>
                </Card>
              ) : null}

              {expensesStatus === "ready" ? (
                <>
                  <div className="lg:min-h-0 lg:flex-1 lg:overflow-y-auto lg:pr-2">
                    <ExpenseList
                      expenses={expensePage.items}
                      deletingId={deletingId}
                      hasFilters={hasFilters}
                      onDelete={handleDelete}
                    />
                  </div>

                  {expensePage.totalCount > 0 ? (
                    <ExpensePagination
                      showingStart={showingStart}
                      showingEnd={showingEnd}
                      totalCount={expensePage.totalCount}
                      offset={offset}
                      pageSize={PAGE_SIZE}
                      onChange={changePage}
                    />
                  ) : null}
                </>
              ) : null}
            </section>
          </div>
        ) : null}
      </div>
    </AppShell>
  )
}

function BackToProjects({ onClick }: { onClick: () => void }) {
  return (
    <Button
      type="button"
      variant="ghost"
      size="sm"
      className="-ml-2 w-fit text-label text-muted-foreground uppercase"
      onClick={onClick}
    >
      <ArrowLeftIcon data-icon="inline-start" />
      Projects
    </Button>
  )
}

function ExpensePagination({
  showingStart,
  showingEnd,
  totalCount,
  offset,
  pageSize,
  onChange,
}: {
  showingStart: number
  showingEnd: number
  totalCount: number
  offset: number
  pageSize: number
  onChange: (offset: number) => void
}) {
  return (
    <nav
      aria-label="Expense pagination"
      className="flex flex-wrap items-center justify-between gap-3"
    >
      <p
        className="font-mono text-[11px] tracking-[0.14em] text-muted-foreground uppercase"
        aria-live="polite"
      >
        Showing {showingStart}–{showingEnd} of {totalCount}
      </p>
      <div className="flex gap-2">
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={offset === 0}
          onClick={() => onChange(Math.max(0, offset - pageSize))}
        >
          Previous
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={offset + pageSize >= totalCount}
          onClick={() => onChange(offset + pageSize)}
        >
          Next
        </Button>
      </div>
    </nav>
  )
}

function projectErrorMessage(error: unknown): string {
  if (error instanceof ApiError && error.status === 403) {
    return "You no longer have access to this project."
  }
  if (error instanceof ApiError && error.status === 404) {
    return "Project not found."
  }
  return orgErrorMessage(error)
}
