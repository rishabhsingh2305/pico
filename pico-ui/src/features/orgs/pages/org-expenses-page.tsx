import { useCallback, useEffect, useState } from "react"
import { Link, useNavigate, useParams } from "react-router"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { CircleAlertIcon, FolderIcon } from "lucide-react"

import { queryKeys } from "@/app/query-keys"
import { PageHeader } from "@/components/page-header"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Spinner } from "@/components/ui/spinner"
import { useAuth, useSession } from "@/features/auth/auth-provider"
import { isSessionInvalidError } from "@/features/auth/session"
import {
  deleteExpense,
  listExpenses,
  summarizeExpenses,
} from "@/features/expenses/api"
import { ExpenseList } from "@/features/expenses/components/expense-list"
import { ExpensePagination } from "@/features/expenses/components/expense-pagination"
import { OrgExpenseFilterForm } from "@/features/expenses/components/org-expense-filter-form"
import { OrgExpenseSummary } from "@/features/expenses/components/expense-summary"
import { expenseErrorMessage } from "@/features/expenses/error-message"
import type { ExpenseFilters } from "@/features/expenses/filters"
import { summarizeByCurrency } from "@/features/expenses/summary"
import type { Expense, ExpensePage } from "@/features/expenses/types"
import { useOrgWorkspace } from "@/features/orgs/layouts/org-workspace-layout"
import { listProjects } from "@/features/projects/api"
import { usePageTitle } from "@/hooks/use-page-title"

const PAGE_SIZE = 20
const EMPTY_PAGE: ExpensePage = { items: [], nextOffset: null }

export function OrgExpensesPage() {
  const { orgId = "" } = useParams<{ orgId: string }>()
  const navigate = useNavigate()
  const { logout } = useAuth()
  const session = useSession()
  const queryClient = useQueryClient()
  const { org } = useOrgWorkspace()

  const [projectId, setProjectId] = useState("")
  const [filters, setFilters] = useState<ExpenseFilters>({ tags: [] })
  const [offset, setOffset] = useState(0)
  const [selectedCurrency, setSelectedCurrency] = useState("")
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [deleteError, setDeleteError] = useState("")

  usePageTitle(`${org.name} expenses · Pico`)

  const projectsQuery = useQuery({
    queryKey: queryKeys.projects(session.userId, orgId, []),
    queryFn: () => listProjects(orgId),
    enabled: orgId !== "",
  })
  const expensesQuery = useQuery({
    queryKey: queryKeys.orgExpenses(
      session.userId,
      orgId,
      projectId,
      filters,
      offset
    ),
    queryFn: () =>
      listExpenses({
        orgId,
        projectId: projectId || undefined,
        tags: filters.tags,
        from: filters.from,
        to: filters.to,
        limit: PAGE_SIZE,
        offset,
      }),
    enabled: orgId !== "",
  })
  const summaryQuery = useQuery({
    queryKey: queryKeys.orgExpenseSummary(
      session.userId,
      orgId,
      projectId,
      filters
    ),
    queryFn: () =>
      summarizeExpenses({
        orgId,
        projectId: projectId || undefined,
        tags: filters.tags,
        from: filters.from,
        to: filters.to,
      }),
    enabled: orgId !== "",
  })

  const sessionInvalid = useCallback(() => {
    void logout()
    navigate("/login", { replace: true })
  }, [logout, navigate])

  useEffect(() => {
    const error =
      projectsQuery.error ?? expensesQuery.error ?? summaryQuery.error
    if (error && isSessionInvalidError(error)) {
      sessionInvalid()
    }
  }, [
    projectsQuery.error,
    expensesQuery.error,
    summaryQuery.error,
    sessionInvalid,
  ])

  const projects = projectsQuery.data ?? []
  const projectNames = Object.fromEntries(
    projects.map((project) => [project.id, project.name])
  )
  const expensePage = expensesQuery.data ?? EMPTY_PAGE
  const summaryItems = summaryQuery.data?.items ?? []
  const currencyTotals = summarizeByCurrency(summaryItems)
  const activeSummary =
    currencyTotals.find((item) => item.currency === selectedCurrency) ??
    currencyTotals[0] ??
    null
  const totalCount = summaryItems.reduce(
    (sum, item) => sum + item.totalCount,
    0
  )
  const hasFilters =
    projectId !== "" ||
    filters.tags.length > 0 ||
    filters.from !== undefined ||
    filters.to !== undefined
  const showingStart = totalCount === 0 ? 0 : offset + 1
  const showingEnd = offset + expensePage.items.length

  async function handleDelete(expense: Expense) {
    setDeletingId(expense.id)
    setDeleteError("")
    try {
      await deleteExpense(orgId, expense.id)
      const nextOffset =
        expensePage.items.length === 1 && offset > 0
          ? Math.max(0, offset - PAGE_SIZE)
          : offset
      if (nextOffset !== offset) {
        setOffset(nextOffset)
      }
      void queryClient.invalidateQueries({
        queryKey: queryKeys.orgExpensesRoot(session.userId, orgId),
      })
    } catch (error) {
      if (isSessionInvalidError(error)) {
        sessionInvalid()
        return
      }
      setDeleteError(expenseErrorMessage(error))
    } finally {
      setDeletingId(null)
    }
  }

  function applyFilters(nextProjectId: string, nextFilters: ExpenseFilters) {
    setProjectId(nextProjectId)
    setFilters(nextFilters)
    setOffset(0)
    setDeleteError("")
  }

  function clearFilters() {
    setProjectId("")
    setFilters({ tags: [] })
    setOffset(0)
    setDeleteError("")
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Organization"
        title="Expenses"
        description={`Every expense across ${org.name}.`}
        actions={
          <Link
            to={`/orgs/${orgId}/projects`}
            className={buttonVariants({
              variant: "outline",
              className:
                "h-10 px-4 text-xs font-semibold tracking-[0.14em] uppercase",
            })}
          >
            <FolderIcon data-icon="inline-start" />
            Projects
          </Link>
        }
      />

      {activeSummary ? (
        <OrgExpenseSummary
          summary={activeSummary}
          currencies={currencyTotals.map((item) => item.currency)}
          selectedCurrency={activeSummary.currency}
          onCurrencyChange={setSelectedCurrency}
        />
      ) : null}

      <OrgExpenseFilterForm
        projects={projects.map((project) => ({
          id: project.id,
          name: project.name,
        }))}
        projectId={projectId}
        onApply={applyFilters}
        onClear={clearFilters}
      />

      {deleteError ? (
        <Alert variant="destructive">
          <CircleAlertIcon aria-hidden="true" />
          <AlertDescription>{deleteError}</AlertDescription>
        </Alert>
      ) : null}

      {expensesQuery.isPending ? (
        <Card>
          <CardContent className="flex items-center justify-center gap-2 py-10 text-sm text-muted-foreground">
            <Spinner />
            Loading expenses...
          </CardContent>
        </Card>
      ) : null}

      {expensesQuery.isError ? (
        <Card>
          <CardContent className="flex flex-col gap-4 py-6">
            <Alert variant="destructive">
              <CircleAlertIcon aria-hidden="true" />
              <AlertDescription>
                {expenseErrorMessage(expensesQuery.error)}
              </AlertDescription>
            </Alert>
            <Button
              type="button"
              variant="outline"
              className="w-fit"
              onClick={() => void expensesQuery.refetch()}
            >
              Try again
            </Button>
          </CardContent>
        </Card>
      ) : null}

      {expensesQuery.isSuccess ? (
        <>
          <ExpenseList
            expenses={expensePage.items}
            deletingId={deletingId}
            hasFilters={hasFilters}
            projectNames={projectNames}
            emptyDescription="Open a project to add the first expense."
            onDelete={(expense) => void handleDelete(expense)}
          />
          {totalCount > 0 ? (
            <ExpensePagination
              showingStart={showingStart}
              showingEnd={showingEnd}
              totalCount={totalCount}
              offset={offset}
              pageSize={PAGE_SIZE}
              nextOffset={expensePage.nextOffset}
              onChange={setOffset}
              label="Organization expense pagination"
            />
          ) : null}
        </>
      ) : null}
    </div>
  )
}
