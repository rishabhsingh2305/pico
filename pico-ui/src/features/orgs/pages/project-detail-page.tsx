import { useCallback, useEffect, useId, useState } from "react"
import { useNavigate, useParams } from "react-router"
import { useQuery, useQueryClient } from "@tanstack/react-query"
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
import { queryKeys } from "@/app/query-keys"
import { useAuth, useSession } from "@/features/auth/auth-provider"
import { isSessionInvalidError, type Session } from "@/features/auth/session"
import { CreateExpenseForm } from "@/features/expenses/components/create-expense-form"
import { ExpenseFilterForm } from "@/features/expenses/components/expense-filter-form"
import { ExpenseList } from "@/features/expenses/components/expense-list"
import {
  deleteExpense,
  listExpenses,
  summarizeExpenses,
} from "@/features/expenses/api"
import { expenseErrorMessage } from "@/features/expenses/error-message"
import type { ExpenseFilters } from "@/features/expenses/filters"
import { formatMoney } from "@/features/expenses/money"
import type { Expense, ExpensePage } from "@/features/expenses/types"
import { getOrg } from "@/features/orgs/api"
import { orgErrorMessage } from "@/features/orgs/error-message"
import { getProject } from "@/features/projects/api"
import { usePageTitle } from "@/hooks/use-page-title"
import { ApiError } from "@/lib/api-client"
import { cn } from "cn"

type LoadStatus = "loading" | "ready" | "error"

const PAGE_SIZE = 20
const EMPTY_PAGE: ExpensePage = {
  items: [],
  nextOffset: null,
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

  const [deleteError, setDeleteError] = useState("")
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [filters, setFilters] = useState<ExpenseFilters>({ tags: [] })
  const [offset, setOffset] = useState(0)
  const [isExpenseDialogOpen, setIsExpenseDialogOpen] = useState(false)
  const queryClient = useQueryClient()

  const orgQuery = useQuery({
    queryKey: queryKeys.org(session.userId, orgId),
    queryFn: () => getOrg(session.userId, orgId),
  })
  const projectQuery = useQuery({
    queryKey: queryKeys.project(session.userId, orgId, projectId),
    queryFn: () => getProject(session.userId, orgId, projectId),
  })
  const expensesQuery = useQuery({
    queryKey: queryKeys.expenses(
      session.userId,
      orgId,
      projectId,
      filters,
      offset
    ),
    queryFn: () =>
      listExpenses({
        userId: session.userId,
        orgId,
        projectId,
        tags: filters.tags,
        from: filters.from,
        to: filters.to,
        limit: PAGE_SIZE,
        offset,
      }),
  })
  const summaryQuery = useQuery({
    queryKey: queryKeys.expenseSummary(
      session.userId,
      orgId,
      projectId,
      filters
    ),
    queryFn: () =>
      summarizeExpenses({
        userId: session.userId,
        orgId,
        projectId,
        tags: filters.tags,
        from: filters.from,
        to: filters.to,
      }),
  })

  const org = orgQuery.data ?? null
  const project = projectQuery.data ?? null
  const expensePage: ExpensePage = expensesQuery.data ?? EMPTY_PAGE
  const summary = summaryQuery.data?.items[0] ?? null
  const status: LoadStatus =
    orgQuery.isPending || projectQuery.isPending
      ? "loading"
      : orgQuery.isError || projectQuery.isError
        ? "error"
        : "ready"
  const error =
    orgQuery.error || projectQuery.error
      ? projectErrorMessage(orgQuery.error ?? projectQuery.error)
      : ""
  const expensesStatus: LoadStatus = expensesQuery.isPending
    ? "loading"
    : expensesQuery.isError
      ? "error"
      : "ready"
  const expensesError = expensesQuery.isError
    ? projectErrorMessage(expensesQuery.error)
    : ""
  const summaryError = summaryQuery.isError
    ? expenseErrorMessage(summaryQuery.error)
    : ""
  const isUpdating = expensesQuery.isFetching && !expensesQuery.isPending

  usePageTitle(project ? `${project.name} · Pico` : "Project · Pico")

  const sessionInvalid = useCallback(() => {
    void logout()
    navigate("/login", { replace: true })
  }, [logout, navigate])

  useEffect(() => {
    const queryError =
      orgQuery.error ??
      projectQuery.error ??
      expensesQuery.error ??
      summaryQuery.error
    if (queryError && isSessionInvalidError(queryError)) {
      sessionInvalid()
    }
  }, [
    orgQuery.error,
    projectQuery.error,
    expensesQuery.error,
    summaryQuery.error,
    sessionInvalid,
  ])

  function retryProject() {
    void Promise.all([orgQuery.refetch(), projectQuery.refetch()])
  }

  function retryExpenses() {
    void Promise.all([expensesQuery.refetch(), summaryQuery.refetch()])
  }

  function handleFiltersApplied(nextFilters: ExpenseFilters) {
    setDeleteError("")
    setFilters(nextFilters)
    setOffset(0)
  }

  function handleFiltersCleared() {
    setDeleteError("")
    setFilters({ tags: [] })
    setOffset(0)
  }

  function handleExpenseChanged() {
    setDeleteError("")
    void queryClient.invalidateQueries({
      queryKey: queryKeys.expenseRoot(session.userId, orgId, projectId),
    })
  }

  function changePage(nextOffset: number) {
    if (nextOffset === offset) {
      return
    }
    setDeleteError("")
    setOffset(nextOffset)
  }

  async function handleDelete(expense: Expense) {
    setDeletingId(expense.id)
    setDeleteError("")
    try {
      await deleteExpense(session.userId, orgId, expense.id)
      const remainingCount = summary ? summary.totalCount - 1 : null
      void queryClient.invalidateQueries({
        queryKey: queryKeys.expenseRoot(session.userId, orgId, projectId),
      })
      if (remainingCount !== null && offset > 0 && remainingCount <= offset) {
        setOffset(Math.max(0, offset - PAGE_SIZE))
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
  const totalCount = summary?.totalCount ?? 0
  const netTotal = summary?.netTotal ?? 0
  const totalCredit = summary?.totalCredit ?? 0
  const totalDebit = summary?.totalDebit ?? 0
  const summaryText = `${totalCount} ${
    totalCount === 1 ? "expense" : "expenses"
  } · ${formatMoney(netTotal, currency)} net · ${formatMoney(
    totalCredit,
    currency
  )} credit · ${formatMoney(totalDebit, currency)} debit`
  const showingStart = totalCount === 0 ? 0 : offset + 1
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
                    netTotal < 0 ? "bg-primary" : "bg-chart-2"
                  )}
                  aria-hidden="true"
                />
                <span
                  className={cn(
                    "absolute inset-x-0 top-0 hidden h-1.5 lg:block",
                    netTotal < 0 ? "bg-primary" : "bg-chart-2"
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
                          {formatMoney(totalCredit, currency)} credit
                        </span>
                        <span className="px-1.5">−</span>
                        <span className="text-primary">
                          {formatMoney(totalDebit, currency)} debit
                        </span>
                      </p>
                      <span className="font-mono text-[10px] tracking-[0.16em] text-muted-foreground uppercase">
                        {currency}
                      </span>
                      {isUpdating ? (
                        <span className="flex items-center gap-1.5 font-mono text-[10px] tracking-[0.12em] text-muted-foreground uppercase">
                          <Spinner className="size-3" />
                          Updating
                        </span>
                      ) : null}
                    </div>
                  </div>

                  <span
                    aria-hidden="true"
                    className={cn(netTotal < 0 && "text-destructive")}
                  >
                    <CountUp
                      value={netTotal}
                      format={(amount) => formatMoney(amount, currency)}
                      className="mt-3 block max-w-full font-display text-metric-fluid whitespace-nowrap tabular-nums"
                      durationMs={800}
                    />
                  </span>

                  <p className="sr-only" aria-live="polite">
                    {summaryText}
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

              {summaryError ? (
                <Alert variant="destructive">
                  <CircleAlertIcon aria-hidden="true" />
                  <AlertDescription className="flex flex-wrap items-center justify-between gap-3">
                    <span>{summaryError}</span>
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={retryExpenses}
                    >
                      Try again
                    </Button>
                  </AlertDescription>
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

                  {totalCount > 0 ? (
                    <ExpensePagination
                      showingStart={showingStart}
                      showingEnd={showingEnd}
                      totalCount={totalCount}
                      offset={offset}
                      pageSize={PAGE_SIZE}
                      nextOffset={expensePage.nextOffset}
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
  nextOffset,
  onChange,
}: {
  showingStart: number
  showingEnd: number
  totalCount: number
  offset: number
  pageSize: number
  nextOffset: number | null
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
          disabled={nextOffset === null}
          onClick={() => {
            if (nextOffset !== null) {
              onChange(nextOffset)
            }
          }}
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
