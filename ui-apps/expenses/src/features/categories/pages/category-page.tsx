import { useCallback, useEffect, useState } from "react"
import { Link, useParams } from "react-router"
import {
  ArrowLeftIcon,
  CircleAlertIcon,
  PlusIcon,
  SlidersHorizontalIcon,
  XIcon,
} from "lucide-react"

import { AppShell } from "@/components/app-shell"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { useAuth } from "@/features/auth/auth-provider"
import { getCategory } from "@/features/categories/api"
import { categoryColor } from "@/features/categories/colors"
import { categoryErrorMessage } from "@/features/categories/error-message"
import type { Category } from "@/features/categories/types"
import { listExpenses } from "@/features/expenses/api"
import { ExpenseForm } from "@/features/expenses/components/expense-form"
import { FilterSheet } from "@/features/expenses/components/filter-sheet"
import {
  activeFilterCount,
  defaultFilters,
  type AppliedFilters,
} from "@/features/expenses/filters"
import { useExpenses } from "@/features/expenses/expenses-provider"
import {
  formatCompactMoney,
  formatExpenseDateShort,
  formatMoney,
} from "@/features/expenses/format"
import {
  dailyDebitTotals,
  monthRange,
  percentChange,
  previousMonthRange,
} from "@/features/expenses/months"
import type { Expense, ExpenseListResponse } from "@/features/expenses/types"
import { usePageTitle } from "@/hooks/use-page-title"
import { ApiError } from "@/lib/api-client"
import { cn } from "@/lib/utils"

type Status = "loading" | "ready" | "error"

type Insight = {
  currentTotal: number
  previousTotal: number
  dailyTotals: number[]
  monthLabel: string
  truncated: boolean
}

export function CategoryPage() {
  const { categoryId = "" } = useParams()
  const { orgId } = useExpenses()
  const { logout } = useAuth()

  const [category, setCategory] = useState<Category | null>(null)
  const [categoryStatus, setCategoryStatus] = useState<Status>("loading")
  const [categoryError, setCategoryError] = useState("")
  const [categoryAttempt, setCategoryAttempt] = useState(0)

  const [expenses, setExpenses] = useState<ExpenseListResponse | null>(null)
  const [expensesStatus, setExpensesStatus] = useState<Status>("loading")
  const [expensesError, setExpensesError] = useState("")
  const [expensesAttempt, setExpensesAttempt] = useState(0)
  const [availableTags, setAvailableTags] = useState<string[]>([])

  const [filters, setFilters] = useState<AppliedFilters>(defaultFilters)
  const [filterSheetOpen, setFilterSheetOpen] = useState(false)
  const [reload, setReload] = useState(0)

  const [isFormOpen, setIsFormOpen] = useState(false)
  const [insight, setInsight] = useState<Insight | null>(null)
  const [insightStatus, setInsightStatus] = useState<
    "loading" | "ready" | "error"
  >("loading")

  const activeCount = activeFilterCount(filters)

  usePageTitle(category ? `${category.name} · Expenses` : "Category · Expenses")

  useEffect(() => {
    if (!categoryId) {
      return
    }

    let cancelled = false

    getCategory(orgId, categoryId)
      .then((nextCategory) => {
        if (!cancelled) {
          setCategory(nextCategory)
          setCategoryStatus("ready")
        }
      })
      .catch(async (caught: unknown) => {
        if (cancelled) {
          return
        }
        if (caught instanceof ApiError && caught.status === 401) {
          await logout()
          return
        }
        setCategoryError(categoryErrorMessage(caught))
        setCategoryStatus("error")
      })

    return () => {
      cancelled = true
    }
  }, [orgId, categoryId, logout, categoryAttempt])

  useEffect(() => {
    if (!categoryId) {
      return
    }

    let cancelled = false

    listExpenses(orgId, categoryId, {
      from: filters.from,
      to: filters.to,
      tags: filters.tags,
      limit: 100,
    })
      .then((response) => {
        if (cancelled) {
          return
        }
        setExpenses(response)
        setExpensesStatus("ready")
        setAvailableTags((current) => {
          const next = new Set(current)
          for (const item of response.items) {
            for (const tag of item.tags) {
              next.add(tag)
            }
          }
          return Array.from(next).sort((a, b) => a.localeCompare(b))
        })
      })
      .catch(async (caught: unknown) => {
        if (cancelled) {
          return
        }
        if (caught instanceof ApiError && caught.status === 401) {
          await logout()
          return
        }
        setExpensesError(categoryErrorMessage(caught))
        setExpensesStatus("error")
      })

    return () => {
      cancelled = true
    }
  }, [orgId, categoryId, filters, logout, expensesAttempt, reload])

  useEffect(() => {
    if (!categoryId) {
      return
    }

    let cancelled = false
    const current = monthRange()
    const previous = previousMonthRange()

    Promise.all([
      listExpenses(orgId, categoryId, {
        from: current.from,
        to: current.to,
        limit: 100,
      }),
      listExpenses(orgId, categoryId, {
        from: previous.from,
        to: previous.to,
        limit: 1,
      }),
    ])
      .then(([currentMonth, previousMonth]) => {
        if (cancelled) {
          return
        }
        setInsight({
          currentTotal: currentMonth.totalDebit,
          previousTotal: previousMonth.totalDebit,
          dailyTotals: dailyDebitTotals(currentMonth.items, current.start),
          monthLabel: current.label,
          truncated: currentMonth.totalCount > currentMonth.items.length,
        })
        setInsightStatus("ready")
      })
      .catch(async (caught: unknown) => {
        if (cancelled) {
          return
        }
        if (caught instanceof ApiError && caught.status === 401) {
          await logout()
          return
        }
        setInsightStatus("error")
      })

    return () => {
      cancelled = true
    }
  }, [orgId, categoryId, logout, reload])

  const retryCategory = useCallback(() => {
    setCategoryStatus("loading")
    setCategoryError("")
    setCategoryAttempt((current) => current + 1)
  }, [])

  const retryExpenses = useCallback(() => {
    setExpensesStatus("loading")
    setExpensesError("")
    setExpensesAttempt((current) => current + 1)
  }, [])

  const handleCreated = useCallback(() => {
    setReload((current) => current + 1)
  }, [])

  function updateFilters(updater: (current: AppliedFilters) => AppliedFilters) {
    setExpensesStatus("loading")
    setFilters(updater)
  }

  function applyFilters(next: AppliedFilters) {
    setExpensesStatus("loading")
    setFilters(next)
    setFilterSheetOpen(false)
  }

  function removeDateFilter() {
    updateFilters((current) => ({ ...defaultFilters, tags: current.tags }))
  }

  function removeTagFilter(tag: string) {
    updateFilters((current) => ({
      ...current,
      tags: current.tags.filter((item) => item !== tag),
    }))
  }

  function clearFilters() {
    updateFilters(() => defaultFilters)
  }

  function toggleTagFilter(tag: string) {
    updateFilters((current) => {
      const hasTag = current.tags.includes(tag)
      return {
        ...current,
        tags: hasTag
          ? current.tags.filter((item) => item !== tag)
          : [...current.tags, tag].sort((a, b) => a.localeCompare(b)),
      }
    })
  }

  return (
    <AppShell>
      <div className="mx-auto flex min-h-0 w-full max-w-5xl flex-1 flex-col">
        <div className="shrink-0 px-4 pt-3 sm:pt-4">
          <div className="flex items-center justify-between gap-3">
            <Link
              to="/"
              className="inline-flex items-center gap-1.5 text-sm font-semibold text-muted-foreground hover:text-foreground"
            >
              <ArrowLeftIcon className="size-4" aria-hidden="true" />
              Your spending
            </Link>

            <Button
              type="button"
              variant="outline"
              size="sm"
              className="h-9"
              onClick={() => setFilterSheetOpen(true)}
              disabled={categoryStatus !== "ready"}
              aria-label={
                activeCount > 0 ? `Filters, ${activeCount} active` : "Filters"
              }
            >
              <SlidersHorizontalIcon className="size-4" aria-hidden="true" />
              Filters
              {activeCount > 0 ? (
                <span className="ml-1 inline-flex size-5 items-center justify-center bg-primary text-[0.65rem] font-bold text-primary-foreground">
                  {activeCount}
                </span>
              ) : null}
            </Button>
          </div>

          {categoryStatus === "ready" && category ? (
            <>
              <section className="mt-3 border-2 border-foreground bg-primary p-4 text-primary-foreground sm:p-8">
                <div className="flex items-start gap-3 sm:gap-4">
                  <span
                    className="mt-1 size-5 shrink-0 border-2 border-primary-foreground sm:size-6"
                    style={{
                      backgroundColor: categoryColor(category.id).background,
                    }}
                    aria-hidden="true"
                  />
                  <div>
                    <h1 className="font-display text-3xl leading-[0.9] tracking-tight sm:text-6xl">
                      {category.name}
                    </h1>
                    <p className="mt-2 text-xs font-medium text-primary-foreground/80 sm:text-sm">
                      Your expenses in {category.name}.
                    </p>
                  </div>
                </div>
              </section>

              <div className="mt-3 grid grid-cols-2 border-2 border-foreground sm:mt-4 sm:grid-cols-3">
                <div className="col-span-2 flex items-center justify-between border-b-2 border-foreground bg-accent p-3 text-accent-foreground sm:col-span-1 sm:block sm:border-r-2 sm:border-b-0 sm:p-5">
                  <span className="text-[0.65rem] font-semibold sm:text-xs">
                    Net
                  </span>
                  <MoneyValue
                    minor={expenses?.netTotal ?? 0}
                    currency={category.currency}
                    className="font-display text-lg leading-none whitespace-nowrap sm:mt-1 sm:block sm:text-3xl"
                  />
                </div>
                <div className="border-r-2 border-foreground bg-destructive p-3 text-destructive-foreground sm:p-5">
                  <span className="text-[0.65rem] font-semibold sm:text-xs">
                    Spent
                  </span>
                  <MoneyValue
                    minor={expenses?.totalDebit ?? 0}
                    currency={category.currency}
                    className="mt-1 block font-display text-sm leading-none whitespace-nowrap sm:text-3xl"
                  />
                </div>
                <div className="bg-card p-3 text-card-foreground sm:p-5">
                  <span className="text-[0.65rem] font-semibold sm:text-xs">
                    Received
                  </span>
                  <MoneyValue
                    minor={expenses?.totalCredit ?? 0}
                    currency={category.currency}
                    className="mt-1 block font-display text-sm leading-none whitespace-nowrap sm:text-3xl"
                  />
                </div>
              </div>
            </>
          ) : null}
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-4 py-4">
          <div className="flex flex-col gap-4">
            {activeCount > 0 ? (
              <ActiveFilters
                filters={filters}
                onRemoveDate={removeDateFilter}
                onRemoveTag={removeTagFilter}
                onClear={clearFilters}
              />
            ) : null}

            {activeCount === 0 &&
            categoryStatus === "ready" &&
            category &&
            insightStatus === "ready" &&
            insight &&
            (insight.currentTotal > 0 || insight.previousTotal > 0) ? (
              <InsightCard insight={insight} currency={category.currency} />
            ) : null}

            {categoryStatus === "error" ? (
              <Alert variant="destructive">
                <CircleAlertIcon aria-hidden="true" />
                <AlertDescription className="flex flex-wrap items-center justify-between gap-3">
                  <span>{categoryError}</span>
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    onClick={retryCategory}
                  >
                    Try again
                  </Button>
                </AlertDescription>
              </Alert>
            ) : null}

            {expensesStatus === "loading" ? <PageSkeleton /> : null}

            {expensesStatus === "error" ? (
              <Alert variant="destructive">
                <CircleAlertIcon aria-hidden="true" />
                <AlertDescription className="flex flex-wrap items-center justify-between gap-3">
                  <span>{expensesError}</span>
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    onClick={retryExpenses}
                  >
                    Try again
                  </Button>
                </AlertDescription>
              </Alert>
            ) : null}

            {expensesStatus === "ready" && expenses ? (
              expenses.items.length === 0 ? (
                activeCount > 0 ? (
                  <EmptyFiltered onClear={clearFilters} />
                ) : (
                  <EmptyExpenses />
                )
              ) : (
                <ExpenseTable
                  items={expenses.items}
                  currency={category?.currency ?? "INR"}
                  selectedTags={filters.tags}
                  onTagClick={toggleTagFilter}
                />
              )
            ) : null}
          </div>
        </div>

        <div className="shrink-0 border-t-2 border-foreground bg-background p-4 pb-[calc(1rem+env(safe-area-inset-bottom))]">
          <div className="flex justify-end">
            <Button
              type="button"
              size="lg"
              className="w-full sm:w-auto"
              onClick={() => setIsFormOpen(true)}
              disabled={categoryStatus !== "ready"}
            >
              <PlusIcon data-icon="inline-start" aria-hidden="true" />
              Add expense
            </Button>
          </div>
        </div>

        {category ? (
          <ExpenseForm
            open={isFormOpen}
            onOpenChange={setIsFormOpen}
            projectId={category.id}
            currency={category.currency}
            onCreated={handleCreated}
          />
        ) : null}

        {filterSheetOpen ? (
          <FilterSheet
            availableTags={availableTags}
            initialFilters={filters}
            onApply={applyFilters}
            onOpenChange={(open) => {
              if (!open) {
                setFilterSheetOpen(false)
              }
            }}
          />
        ) : null}
      </div>
    </AppShell>
  )
}

function ActiveFilters({
  filters,
  onRemoveDate,
  onRemoveTag,
  onClear,
}: {
  filters: AppliedFilters
  onRemoveDate: () => void
  onRemoveTag: (tag: string) => void
  onClear: () => void
}) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      {filters.preset !== "all" ? (
        <FilterChip label={filters.dateLabel} onRemove={onRemoveDate} />
      ) : null}
      {filters.tags.map((tag) => (
        <FilterChip key={tag} label={tag} onRemove={() => onRemoveTag(tag)} />
      ))}
      <Button
        type="button"
        variant="link"
        size="sm"
        className="h-auto px-0 text-xs"
        onClick={onClear}
      >
        Clear all
      </Button>
    </div>
  )
}

function FilterChip({
  label,
  onRemove,
}: {
  label: string
  onRemove: () => void
}) {
  return (
    <span className="inline-flex items-center gap-1 border border-foreground bg-card px-2 py-1 text-xs font-semibold">
      {label}
      <button
        type="button"
        onClick={onRemove}
        aria-label={`Remove ${label} filter`}
        className="p-0.5 text-muted-foreground hover:text-foreground"
      >
        <XIcon className="size-3.5" aria-hidden="true" />
      </button>
    </span>
  )
}

function ExpenseTable({
  items,
  currency,
  selectedTags,
  onTagClick,
}: {
  items: Expense[]
  currency: string
  selectedTags: string[]
  onTagClick: (tag: string) => void
}) {
  return (
    <div className="border-2 border-foreground bg-card">
      <div className="hidden grid-cols-[4.5rem_1fr_auto] items-center gap-3 border-b-2 border-foreground bg-muted px-4 py-2 text-xs font-semibold sm:grid">
        <span>Date</span>
        <span>Description</span>
        <span className="text-right">Amount</span>
      </div>
      {items.map((expense) => (
        <div
          key={expense.id}
          className="flex items-start justify-between gap-4 border-b-2 border-foreground px-4 py-3.5 last:border-b-0 sm:grid sm:grid-cols-[4.5rem_1fr_auto] sm:items-center"
        >
          <span className="hidden w-14 shrink-0 text-xs text-muted-foreground sm:block">
            {formatExpenseDateShort(expense.spentAt)}
          </span>
          <div className="flex min-w-0 flex-col gap-1">
            <span className="truncate text-sm font-medium">
              {expense.description ||
                (expense.type === "debit" ? "Expense" : "Money received")}
            </span>
            <div className="flex flex-wrap items-center gap-1.5">
              <span className="text-xs text-muted-foreground sm:hidden">
                {formatExpenseDateShort(expense.spentAt)}
              </span>
              {expense.tags.map((tag) => {
                const selected = selectedTags.includes(tag)
                return (
                  <button
                    key={tag}
                    type="button"
                    onClick={() => onTagClick(tag)}
                    aria-pressed={selected}
                    className={cn(
                      "border px-1.5 py-0.5 text-[0.7rem] font-semibold transition-colors",
                      selected
                        ? "border-foreground bg-foreground text-background"
                        : "border-foreground/30 bg-muted/50 text-muted-foreground hover:bg-muted hover:text-foreground"
                    )}
                  >
                    {tag}
                  </button>
                )
              })}
            </div>
          </div>
          <span
            className={cn(
              "shrink-0 text-right text-sm font-bold tabular-nums",
              expense.type === "debit" ? "text-spend" : "text-credit"
            )}
          >
            {expense.type === "debit" ? "−" : "+"}
            {formatMoney(expense.amount, currency)}
          </span>
        </div>
      ))}
    </div>
  )
}

function InsightCard({
  insight,
  currency,
}: {
  insight: Insight
  currency: string
}) {
  const change = percentChange(insight.currentTotal, insight.previousTotal)
  const max = Math.max(...insight.dailyTotals, 1)

  return (
    <div className="border-2 border-foreground bg-card p-4 sm:p-5">
      <div className="flex items-start justify-between gap-4">
        <div>
          <span className="text-xs font-semibold text-muted-foreground">
            {insight.monthLabel}
          </span>
          <MoneyValue
            minor={insight.currentTotal}
            currency={currency}
            className="mt-1 block font-display text-3xl leading-none"
          />
        </div>
        <div className="text-right">
          <span className="text-xs font-semibold text-muted-foreground">
            vs last month
          </span>
          <span
            className={cn(
              "mt-1 block font-display text-lg leading-none",
              change === null
                ? "text-muted-foreground"
                : change > 0
                  ? "text-destructive"
                  : change < 0
                    ? "text-primary"
                    : "text-muted-foreground"
            )}
          >
            {change === null
              ? "New"
              : change === 0
                ? "No change"
                : `${change > 0 ? "↑" : "↓"} ${Math.abs(change).toFixed(0)}%`}
          </span>
          <MoneyValue
            minor={insight.previousTotal}
            currency={currency}
            className="mt-1 block text-[0.65rem] text-muted-foreground"
          />
        </div>
      </div>

      <div className="mt-4 flex h-16 items-end gap-0.5" aria-hidden="true">
        {insight.dailyTotals.map((total, index) => {
          const height = max > 0 ? Math.max(4, (total / max) * 100) : 4
          const isMax = total === max && max > 0
          return (
            <div
              key={index}
              className={cn(
                "flex-1",
                isMax ? "bg-destructive" : total > 0 ? "bg-primary" : "bg-muted"
              )}
              style={{ height: `${height}%` }}
            />
          )
        })}
      </div>

      {insight.truncated ? (
        <p className="mt-2 text-[0.65rem] text-muted-foreground">
          Trend based on the latest 100 expenses.
        </p>
      ) : null}
    </div>
  )
}

function MoneyValue({
  minor,
  currency,
  className,
}: {
  minor: number
  currency: string
  className?: string
}) {
  const full = formatMoney(minor, currency)
  return (
    <span className={className} title={full} aria-label={full}>
      {formatCompactMoney(minor, currency)}
    </span>
  )
}

function EmptyExpenses() {
  return (
    <div className="flex flex-col items-center justify-center border-2 border-foreground bg-card px-6 py-12 text-center sm:py-16">
      <span
        className="font-display text-7xl leading-none sm:text-8xl"
        aria-hidden="true"
      >
        0
      </span>
      <div className="mt-4 h-2 w-16 bg-primary" aria-hidden="true" />
      <h2 className="mt-4 font-display text-2xl tracking-tight">
        No expenses yet
      </h2>
      <p className="mt-1 text-sm text-muted-foreground">
        Add your first expense to start tracking.
      </p>
    </div>
  )
}

function EmptyFiltered({ onClear }: { onClear: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center border-2 border-foreground bg-card px-6 py-12 text-center sm:py-16">
      <span
        className="font-display text-7xl leading-none sm:text-8xl"
        aria-hidden="true"
      >
        0
      </span>
      <div className="mt-4 h-2 w-16 bg-primary" aria-hidden="true" />
      <h2 className="mt-4 font-display text-2xl tracking-tight">
        No expenses match these filters
      </h2>
      <p className="mt-1 text-sm text-muted-foreground">
        Try a different date range or fewer tags.
      </p>
      <Button
        type="button"
        variant="outline"
        className="mt-4"
        onClick={onClear}
      >
        Clear filters
      </Button>
    </div>
  )
}

function PageSkeleton() {
  return (
    <div className="flex flex-col gap-4">
      <Skeleton className="h-20 border-2 border-foreground bg-muted" />
      <Skeleton className="h-32 border-2 border-foreground bg-muted" />
    </div>
  )
}
