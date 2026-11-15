import { useCallback, useEffect, useRef, useState, type UIEvent } from "react"
import { Link, useParams } from "react-router"
import {
  ArrowLeftIcon,
  CircleAlertIcon,
  PlusIcon,
  ReceiptTextIcon,
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
import { CategoryGlyph } from "@/features/categories/icons"
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
  formatExpenseDate,
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
  const categoryBlock = category ? categoryColor(category.id) : null
  const flowTotal = (summary?.totalDebit ?? 0) + (summary?.totalCredit ?? 0)
  const spentShare =
    flowTotal > 0 ? ((summary?.totalDebit ?? 0) / flowTotal) * 100 : 0
  const receivedShare =
    flowTotal > 0 ? ((summary?.totalCredit ?? 0) / flowTotal) * 100 : 0

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
              className="inline-flex items-center gap-2 text-sm font-semibold text-muted-foreground transition-colors hover:text-foreground"
            >
              <ArrowLeftIcon className="size-4" aria-hidden="true" />
              All categories
            </Link>

            <Button
              type="button"
              variant="outline"
              size="sm"
              className="h-9 rounded-xl border-border bg-card px-3 text-foreground hover:bg-muted"
              onClick={() => setFilterSheetOpen(true)}
              disabled={categoryStatus !== "ready"}
              aria-label={
                activeCount > 0 ? `Filters, ${activeCount} active` : "Filters"
              }
            >
              <SlidersHorizontalIcon className="size-4" aria-hidden="true" />
              Filters
              {activeCount > 0 ? (
                <span className="ml-1 inline-flex size-5 items-center justify-center rounded-full bg-primary text-[0.65rem] font-bold text-primary-foreground">
                  {activeCount}
                </span>
              ) : null}
            </Button>
          </div>

          {categoryStatus === "ready" && category ? (
            <>
              <section className="mt-4 flex items-center gap-3.5">
                {categoryBlock ? (
                  <span
                    className="flex size-12 shrink-0 items-center justify-center rounded-xl shadow-[inset_0_1px_0_rgba(255,255,255,0.7),0_1px_2px_rgba(16,24,40,0.08)]"
                    style={{
                      backgroundColor: categoryBlock.background,
                      color: categoryBlock.foreground,
                    }}
                  >
                    <CategoryGlyph
                      name={category.name}
                      id={category.id}
                      className="size-5"
                    />
                  </span>
                ) : null}
                <div className="min-w-0">
                  <p className="text-[10px] font-semibold tracking-[0.16em] text-muted-foreground uppercase">
                    Category
                  </p>
                  <h1 className="mt-0.5 font-display text-4xl leading-none tracking-tight text-foreground sm:text-5xl">
                    {category.name}
                  </h1>
                  <p className="mt-1.5 text-xs text-muted-foreground">
                    Your expenses in {category.name}.
                  </p>
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
                    <div className="border-y border-border py-4">
                      <div className="flex items-start justify-between gap-4">
                        <div className="min-w-0">
                          <span className="block text-[10px] font-semibold tracking-[0.14em] text-muted-foreground uppercase">
                            Net balance
                          </span>
                          <MoneyValue
                            minor={summary?.netTotal ?? 0}
                            currency={category.currency}
                            className={cn(
                              "mt-1 block truncate text-3xl leading-none font-semibold tracking-tight tabular-nums",
                              (summary?.netTotal ?? 0) < 0
                                ? "text-spend"
                                : (summary?.netTotal ?? 0) > 0
                                  ? "text-credit"
                                  : "text-foreground"
                            )}
                          />
                        </div>
                        <span className="shrink-0 rounded-full bg-muted px-2.5 py-1 text-[10px] font-semibold tracking-[0.1em] text-muted-foreground uppercase">
                          {activeCount > 0 ? "Filtered" : "All time"}
                        </span>
                      </div>
                      <div className="mt-4 flex h-1 overflow-hidden rounded-full bg-muted">
                        <span
                          className="h-full bg-spend transition-[width] duration-500"
                          style={{ width: `${spentShare}%` }}
                        />
                        <span
                          className="h-full bg-credit transition-[width] duration-500"
                          style={{ width: `${receivedShare}%` }}
                        />
                      </div>
                      <div className="mt-4 grid grid-cols-2 divide-x divide-border border-t border-border pt-3">
                        <div className="min-w-0 pr-3">
                          <div className="flex items-center gap-1.5 text-[10px] font-semibold tracking-[0.12em] text-muted-foreground uppercase">
                            <span className="size-1.5 rounded-full bg-spend" />
                            Spent
                          </div>
                          <MoneyValue
                            minor={summary?.totalDebit ?? 0}
                            currency={category.currency}
                            className="mt-1 block truncate text-lg font-semibold text-foreground tabular-nums"
                          />
                        </div>
                        <div className="min-w-0 pl-3">
                          <div className="flex items-center gap-1.5 text-[10px] font-semibold tracking-[0.12em] text-muted-foreground uppercase">
                            <span className="size-1.5 rounded-full bg-credit" />
                            Received
                          </div>
                          <MoneyValue
                            minor={summary?.totalCredit ?? 0}
                            currency={category.currency}
                            className="mt-1 block truncate text-lg font-semibold text-foreground tabular-nums"
                          />
                        </div>
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
                    <div className="flex items-center gap-3 border-y border-border bg-background/95 py-2">
                      <div className="min-w-0 flex-1">
                        <span className="flex items-center gap-1 text-[9px] font-semibold tracking-[0.1em] text-muted-foreground uppercase">
                          Net
                        </span>
                        <MoneyValue
                          minor={summary?.netTotal ?? 0}
                          currency={category.currency}
                          className={cn(
                            "mt-0.5 block truncate text-sm font-semibold tabular-nums",
                            (summary?.netTotal ?? 0) < 0
                              ? "text-spend"
                              : (summary?.netTotal ?? 0) > 0
                                ? "text-credit"
                                : "text-foreground"
                          )}
                        />
                      </div>
                      <div className="min-w-0 flex-1">
                        <span className="flex items-center gap-1 text-[9px] font-semibold tracking-[0.1em] text-muted-foreground uppercase">
                          <span className="size-1 rounded-full bg-spend" />
                          Spent
                        </span>
                        <MoneyValue
                          minor={summary?.totalDebit ?? 0}
                          currency={category.currency}
                          className="mt-0.5 block truncate text-sm font-semibold text-foreground tabular-nums"
                        />
                      </div>
                      <div className="min-w-0 flex-1">
                        <span className="flex items-center gap-1 text-[9px] font-semibold tracking-[0.1em] text-muted-foreground uppercase">
                          <span className="size-1 rounded-full bg-credit" />
                          Received
                        </span>
                        <MoneyValue
                          minor={summary?.totalCredit ?? 0}
                          currency={category.currency}
                          className="mt-0.5 block truncate text-sm font-semibold text-foreground tabular-nums"
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

        <div className="shrink-0 border-t border-border bg-background p-4 pb-[calc(1rem+env(safe-area-inset-bottom))]">
          <Button
            type="button"
            size="lg"
            className="h-14 w-full rounded-xl text-base"
            onClick={() => setIsFormOpen(true)}
            disabled={categoryStatus !== "ready"}
          >
            <PlusIcon data-icon="inline-start" aria-hidden="true" />
            Add expense
          </Button>
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
    <span className="inline-flex items-center gap-1 rounded-full border border-border bg-card px-3 py-1 text-xs font-semibold text-foreground">
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
  const groups = groupExpensesByDate(items)

  return (
    <div className="flex flex-col gap-6">
      {groups.map((group) => {
        const dailySpend = group.items
          .filter((expense) => expense.type === "debit")
          .reduce((total, expense) => total + expense.amount, 0)

        return (
          <section key={group.key}>
            <div className="flex items-baseline justify-between gap-4 border-b border-border pb-2">
              <h3 className="text-xs font-semibold text-foreground">
                {formatExpenseDate(group.items[0].spentAt)}
              </h3>
              {dailySpend > 0 ? (
                <span className="shrink-0 text-xs font-semibold text-spend tabular-nums">
                  −{formatMoney(dailySpend, currency)}
                </span>
              ) : null}
            </div>

            <ul className="divide-y divide-border">
              {group.items.map((expense) => (
                <li
                  key={expense.id}
                  className="flex items-start justify-between gap-4 py-3.5"
                >
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium text-foreground">
                      {expense.description ||
                        (expense.type === "debit"
                          ? "Expense"
                          : "Money received")}
                    </p>
                    <div className="mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-1.5">
                      <span className="text-[11px] text-muted-foreground">
                        {formatExpenseDateShort(expense.spentAt)}
                      </span>
                      {expense.tags.map((tag) => {
                        const selected = selectedTags.includes(tag)
                        const block = categoryColor(tag)
                        return (
                          <button
                            key={tag}
                            type="button"
                            onClick={() => onTagClick(tag)}
                            aria-pressed={selected}
                            className={cn(
                              "rounded-full border px-2 py-0.5 text-[10px] font-medium tracking-wide transition-colors",
                              selected
                                ? "border-primary bg-primary text-primary-foreground"
                                : "hover:opacity-80"
                            )}
                            style={
                              selected
                                ? undefined
                                : {
                                    backgroundColor: block.background,
                                    borderColor: block.border,
                                    color: block.foreground,
                                  }
                            }
                          >
                            {tag}
                          </button>
                        )
                      })}
                    </div>
                  </div>
                  <span
                    className={cn(
                      "shrink-0 text-right text-[15px] font-semibold tabular-nums",
                      expense.type === "debit" ? "text-spend" : "text-credit"
                    )}
                  >
                    {expense.type === "debit" ? "−" : "+"}
                    {formatMoney(expense.amount, currency)}
                  </span>
                </li>
              ))}
            </ul>
          </section>
        )
      })}
    </div>
  )
}

type ExpenseGroup = {
  key: string
  items: Expense[]
}

function groupExpensesByDate(items: Expense[]): ExpenseGroup[] {
  const groups = new Map<string, Expense[]>()

  for (const expense of items) {
    const date = new Date(expense.spentAt)
    const key = Number.isNaN(date.getTime())
      ? expense.spentAt
      : `${date.getFullYear()}-${date.getMonth()}-${date.getDate()}`
    const group = groups.get(key)
    if (group) {
      group.push(expense)
    } else {
      groups.set(key, [expense])
    }
  }

  return Array.from(groups, ([key, groupItems]) => ({
    key,
    items: groupItems,
  }))
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
    <div className="flex flex-col items-center justify-center border-y border-border px-6 py-14 text-center">
      <span
        className="flex size-12 items-center justify-center rounded-full bg-muted text-muted-foreground"
        aria-hidden="true"
      >
        <ReceiptTextIcon className="size-5" />
      </span>
      <h2 className="mt-4 font-display text-2xl text-foreground">
        No expenses yet
      </h2>
      <p className="mt-1 max-w-xs text-sm text-muted-foreground">
        Add your first expense to start tracking.
      </p>
    </div>
  )
}

function EmptyFiltered({ onClear }: { onClear: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center border-y border-border px-6 py-14 text-center">
      <span
        className="flex size-12 items-center justify-center rounded-full bg-muted text-muted-foreground"
        aria-hidden="true"
      >
        <ReceiptTextIcon className="size-5" />
      </span>
      <h2 className="mt-4 font-display text-2xl text-foreground">
        No expenses match these filters
      </h2>
      <p className="mt-1 max-w-xs text-sm text-muted-foreground">
        Try a different date range or fewer tags.
      </p>
      <Button
        type="button"
        variant="outline"
        className="mt-4 rounded-xl"
        onClick={onClear}
      >
        Clear filters
      </Button>
    </div>
  )
}

function PageSkeleton() {
  return (
    <div className="flex flex-col gap-3">
      <Skeleton className="h-5 w-24 bg-muted" />
      <Skeleton className="h-12 w-full bg-muted" />
      <Skeleton className="h-12 w-full bg-muted" />
      <Skeleton className="h-12 w-full bg-muted" />
    </div>
  )
}
