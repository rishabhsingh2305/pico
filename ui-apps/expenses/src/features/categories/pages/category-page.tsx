import { useCallback, useEffect, useRef, useState, type UIEvent } from "react"
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
import { listExpenses, summarizeExpenses } from "@/features/expenses/api"
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
import type { Expense, ExpenseSummaryItem } from "@/features/expenses/types"
import { usePageTitle } from "@/hooks/use-page-title"
import { ApiError } from "@/lib/api-client"
import { cn } from "@/lib/utils"

type Status = "loading" | "ready" | "error"

export function CategoryPage() {
  const { categoryId = "" } = useParams()
  const { orgId } = useExpenses()
  const { logout } = useAuth()

  const [category, setCategory] = useState<Category | null>(null)
  const [categoryStatus, setCategoryStatus] = useState<Status>("loading")
  const [categoryError, setCategoryError] = useState("")
  const [categoryAttempt, setCategoryAttempt] = useState(0)

  const [expenses, setExpenses] = useState<Expense[] | null>(null)
  const [summary, setSummary] = useState<ExpenseSummaryItem | null>(null)
  const [expensesStatus, setExpensesStatus] = useState<Status>("loading")
  const [expensesError, setExpensesError] = useState("")
  const [expensesAttempt, setExpensesAttempt] = useState(0)
  const [availableTags, setAvailableTags] = useState<string[]>([])

  const [filters, setFilters] = useState<AppliedFilters>(defaultFilters)
  const [filterSheetOpen, setFilterSheetOpen] = useState(false)
  const [reload, setReload] = useState(0)
  const [totalsCollapsed, setTotalsCollapsed] = useState(false)
  const listRef = useRef<HTMLDivElement>(null)
  const lastScrollTop = useRef(0)
  const lastListHeight = useRef(0)
  const scrollAccumulator = useRef(0)
  const ignoreScrollUntil = useRef(0)
  const scrollFrame = useRef<number | null>(null)
  const scrollLockTimer = useRef<number | null>(null)
  const totalsCollapsedRef = useRef(false)

  const [isFormOpen, setIsFormOpen] = useState(false)

  const activeCount = activeFilterCount(filters)

  usePageTitle(category ? `${category.name} · Expenses` : "Category · Expenses")

  useEffect(() => {
    return () => {
      if (scrollFrame.current !== null) {
        window.cancelAnimationFrame(scrollFrame.current)
      }
      if (scrollLockTimer.current !== null) {
        window.clearTimeout(scrollLockTimer.current)
      }
    }
  }, [])

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
        setExpenses(response.items)
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

    summarizeExpenses(orgId, categoryId, {
      from: filters.from,
      to: filters.to,
      tags: filters.tags,
    })
      .then((response) => {
        if (!cancelled) {
          setSummary(response.items[0] ?? null)
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
        setSummary(null)
      })

    return () => {
      cancelled = true
    }
  }, [orgId, categoryId, filters, logout, expensesAttempt, reload])

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

  function handleCreated() {
    resetListPosition()
    setReload((current) => current + 1)
  }

  function setCollapsed(next: boolean) {
    if (totalsCollapsedRef.current === next) {
      return
    }
    totalsCollapsedRef.current = next
    scrollAccumulator.current = 0
    ignoreScrollUntil.current = window.performance.now() + 260
    setTotalsCollapsed(next)

    if (scrollLockTimer.current !== null) {
      window.clearTimeout(scrollLockTimer.current)
    }
    scrollLockTimer.current = window.setTimeout(() => {
      if (listRef.current) {
        lastScrollTop.current = listRef.current.scrollTop
        lastListHeight.current = listRef.current.clientHeight
      }
      scrollAccumulator.current = 0
    }, 260)
  }

  function resetListPosition() {
    if (listRef.current) {
      listRef.current.scrollTop = 0
      lastListHeight.current = listRef.current.clientHeight
    }
    if (scrollLockTimer.current !== null) {
      window.clearTimeout(scrollLockTimer.current)
      scrollLockTimer.current = null
    }
    ignoreScrollUntil.current = 0
    lastScrollTop.current = 0
    scrollAccumulator.current = 0
    totalsCollapsedRef.current = false
    setTotalsCollapsed(false)
  }

  function processScroll(element: HTMLDivElement) {
    const top = element.scrollTop
    const height = element.clientHeight

    // A header height transition changes the scroll container height. Treat the
    // resulting scroll event as layout movement, not user intent.
    if (
      lastListHeight.current !== 0 &&
      Math.abs(height - lastListHeight.current) > 4
    ) {
      lastListHeight.current = height
      lastScrollTop.current = top
      scrollAccumulator.current = 0
      return
    }
    lastListHeight.current = height

    if (window.performance.now() < ignoreScrollUntil.current) {
      lastScrollTop.current = top
      return
    }

    const delta = top - lastScrollTop.current
    lastScrollTop.current = top

    if (top <= 16) {
      scrollAccumulator.current = 0
      setCollapsed(false)
      return
    }
    if (delta === 0) {
      return
    }

    if (delta > 0 !== scrollAccumulator.current > 0) {
      scrollAccumulator.current = 0
    }
    scrollAccumulator.current += delta

    if (scrollAccumulator.current > 32 && top > 40) {
      setCollapsed(true)
    } else if (scrollAccumulator.current < -32) {
      setCollapsed(false)
    }
  }

  function handleListScroll(event: UIEvent<HTMLDivElement>) {
    const element = event.currentTarget
    if (scrollFrame.current !== null) {
      return
    }
    scrollFrame.current = window.requestAnimationFrame(() => {
      scrollFrame.current = null
      processScroll(element)
    })
  }

  function updateFilters(updater: (current: AppliedFilters) => AppliedFilters) {
    setExpensesStatus("loading")
    setFilters(updater)
    resetListPosition()
  }

  function applyFilters(next: AppliedFilters) {
    setExpensesStatus("loading")
    setFilters(next)
    setFilterSheetOpen(false)
    resetListPosition()
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

              <div className="mt-3 sm:mt-4">
                <div
                  aria-hidden={totalsCollapsed}
                  className={cn(
                    "grid transition-[grid-template-rows,opacity] duration-200 ease-out motion-reduce:transition-none",
                    totalsCollapsed
                      ? "grid-rows-[0fr] opacity-0"
                      : "grid-rows-[1fr] opacity-100"
                  )}
                >
                  <div className="min-h-0 overflow-hidden">
                    <div className="grid grid-cols-2 border-2 border-foreground sm:grid-cols-3">
                      <div className="col-span-2 flex items-center justify-between border-b-2 border-foreground bg-accent p-3 text-accent-foreground sm:col-span-1 sm:block sm:border-r-2 sm:border-b-0 sm:p-5">
                        <span className="text-[0.65rem] font-semibold sm:text-xs">
                          Net
                        </span>
                        <MoneyValue
                          minor={summary?.netTotal ?? 0}
                          currency={category.currency}
                          className="font-display text-lg leading-none whitespace-nowrap sm:mt-1 sm:block sm:text-3xl"
                        />
                      </div>
                      <div className="border-r-2 border-foreground bg-destructive p-3 text-destructive-foreground sm:p-5">
                        <span className="text-[0.65rem] font-semibold sm:text-xs">
                          Spent
                        </span>
                        <MoneyValue
                          minor={summary?.totalDebit ?? 0}
                          currency={category.currency}
                          className="mt-1 block font-display text-sm leading-none whitespace-nowrap sm:text-3xl"
                        />
                      </div>
                      <div className="bg-card p-3 text-card-foreground sm:p-5">
                        <span className="text-[0.65rem] font-semibold sm:text-xs">
                          Received
                        </span>
                        <MoneyValue
                          minor={summary?.totalCredit ?? 0}
                          currency={category.currency}
                          className="mt-1 block font-display text-sm leading-none whitespace-nowrap sm:text-3xl"
                        />
                      </div>
                    </div>
                  </div>
                </div>

                <div
                  aria-hidden={!totalsCollapsed}
                  className={cn(
                    "grid transition-[grid-template-rows,opacity] duration-200 ease-out motion-reduce:transition-none",
                    totalsCollapsed
                      ? "grid-rows-[1fr] opacity-100"
                      : "grid-rows-[0fr] opacity-0"
                  )}
                >
                  <div className="min-h-0 overflow-hidden">
                    <div className="flex border-2 border-foreground">
                      <div className="min-w-0 flex-1 border-r-2 border-foreground bg-accent px-2.5 py-2 text-accent-foreground">
                        <span className="block text-[0.55rem] font-semibold uppercase opacity-80">
                          Net
                        </span>
                        <MoneyValue
                          minor={summary?.netTotal ?? 0}
                          currency={category.currency}
                          className="mt-0.5 block truncate font-display text-sm leading-none"
                        />
                      </div>
                      <div className="min-w-0 flex-1 border-r-2 border-foreground bg-destructive px-2.5 py-2 text-destructive-foreground">
                        <span className="block text-[0.55rem] font-semibold uppercase opacity-80">
                          Spent
                        </span>
                        <MoneyValue
                          minor={summary?.totalDebit ?? 0}
                          currency={category.currency}
                          className="mt-0.5 block truncate font-display text-sm leading-none"
                        />
                      </div>
                      <div className="min-w-0 flex-1 bg-card px-2.5 py-2 text-card-foreground">
                        <span className="block text-[0.55rem] font-semibold uppercase opacity-80">
                          Received
                        </span>
                        <MoneyValue
                          minor={summary?.totalCredit ?? 0}
                          currency={category.currency}
                          className="mt-0.5 block truncate font-display text-sm leading-none"
                        />
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </>
          ) : null}
        </div>

        <div
          ref={listRef}
          onScroll={handleListScroll}
          className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-4 py-4 [overflow-anchor:none]"
        >
          <div className="flex flex-col gap-4">
            {activeCount > 0 ? (
              <ActiveFilters
                filters={filters}
                onRemoveDate={removeDateFilter}
                onRemoveTag={removeTagFilter}
                onClear={clearFilters}
              />
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
              expenses.length === 0 ? (
                activeCount > 0 ? (
                  <EmptyFiltered onClear={clearFilters} />
                ) : (
                  <EmptyExpenses />
                )
              ) : (
                <ExpenseTable
                  items={expenses}
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
