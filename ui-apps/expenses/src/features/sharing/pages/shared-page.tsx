import { useCallback, useEffect, useRef, useState } from "react"
import { GlobeIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Spinner } from "@/components/ui/spinner"
import { getCategory } from "@/features/categories/api"
import type { Category } from "@/features/categories/types"
import {
  getExpense,
  listExpenses,
  summarizeExpenses,
} from "@/features/expenses/api"
import { formatExpenseDate, formatMoney } from "@/features/expenses/format"
import {
  summarizeByCurrency,
  totalsFromExpense,
  type CurrencyTotals,
} from "@/features/expenses/summary"
import type { Expense } from "@/features/expenses/types"
import { cn } from "@/lib/utils"
import {
  acquireShareLink,
  clearShareToken,
  setShareToken,
  type ShareLink,
} from "@/lib/share-token"

type SharedData = {
  category: Category | null
  expenses: Expense[]
  summary: CurrencyTotals[]
  nextOffset: number | null
}

type SharedStatus = "loading" | "ready" | "error"

const SHARED_PAGE_SIZE = 20

export function SharedPage() {
  const [link] = useState<ShareLink | null>(acquireShareLink)
  const [data, setData] = useState<SharedData | null>(null)
  const [status, setStatus] = useState<SharedStatus>("loading")
  const [loadingMore, setLoadingMore] = useState(false)
  const [loadMoreError, setLoadMoreError] = useState("")
  const loadingMoreRef = useRef(false)
  const mountedRef = useRef(true)
  const sentinelRef = useRef<HTMLDivElement>(null)
  const projectId = link
    ? (link.projectId ??
      (link.resourceType === "project" ? link.resourceId : ""))
    : ""

  useEffect(() => {
    mountedRef.current = true
    return () => {
      mountedRef.current = false
    }
  }, [])

  useEffect(() => {
    if (!link) {
      return
    }
    setShareToken(link.token)
    let cancelled = false
    loadShared(link, projectId)
      .then((result) => {
        if (!cancelled) {
          setData(result)
          setStatus("ready")
        }
      })
      .catch(() => {
        if (!cancelled) {
          setStatus("error")
        }
      })
    return () => {
      cancelled = true
      clearShareToken()
    }
  }, [link, projectId])

  const loadMore = useCallback(async () => {
    if (!link || !data || data.nextOffset === null || loadingMoreRef.current) {
      return
    }

    loadingMoreRef.current = true
    setLoadingMore(true)
    setLoadMoreError("")
    try {
      const page = await listExpenses(link.orgId, projectId || undefined, {
        limit: SHARED_PAGE_SIZE,
        offset: data.nextOffset,
      })
      if (!mountedRef.current) {
        return
      }
      setData((current) => {
        if (!current) {
          return current
        }
        const existingIDs = new Set(
          current.expenses.map((expense) => expense.id)
        )
        return {
          ...current,
          expenses: [
            ...current.expenses,
            ...page.items.filter((expense) => !existingIDs.has(expense.id)),
          ],
          nextOffset: page.nextOffset,
        }
      })
    } catch {
      if (mountedRef.current) {
        setLoadMoreError("Could not load more expenses.")
      }
    } finally {
      loadingMoreRef.current = false
      if (mountedRef.current) {
        setLoadingMore(false)
      }
    }
  }, [data, link, projectId])

  useEffect(() => {
    if (
      !data ||
      data.nextOffset === null ||
      loadingMore ||
      loadMoreError !== ""
    ) {
      return
    }
    const sentinel = sentinelRef.current
    if (!sentinel || typeof IntersectionObserver === "undefined") {
      return
    }

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) {
          void loadMore()
        }
      },
      { rootMargin: "200px 0px" }
    )
    observer.observe(sentinel)
    return () => observer.disconnect()
  }, [data, loadMore, loadingMore, loadMoreError])

  if (!link) {
    return <Unavailable message="This share link is missing its token." />
  }
  if (status === "error") {
    return <Unavailable message="This link is no longer available." />
  }
  if (status === "loading" || !data) {
    return (
      <div className="flex min-h-svh items-center justify-center">
        <Spinner className="size-6 text-muted-foreground" />
      </div>
    )
  }

  const title =
    data.category?.name ??
    (link.resourceType === "expense" ? "Shared expense" : "Shared expenses")

  return (
    <main className="mx-auto flex w-full max-w-3xl flex-col gap-6 px-4 py-8">
      <header className="flex flex-col gap-2 border-b border-border pb-5">
        <span className="flex items-center gap-2 text-[10px] font-semibold tracking-[0.16em] text-muted-foreground uppercase">
          <GlobeIcon className="size-3.5" aria-hidden="true" />
          Shared read-only view
        </span>
        <h1 className="font-display text-4xl leading-none tracking-tight">
          {title}
        </h1>
        <p className="text-sm text-muted-foreground">
          Anyone with this link can view these expenses. No account required.
        </p>
      </header>

      {data.summary.length > 0 ? <Summary summary={data.summary} /> : null}

      {data.expenses.length === 0 ? (
        <Card>
          <CardContent className="py-10 text-center text-sm text-muted-foreground">
            No expenses to show.
          </CardContent>
        </Card>
      ) : (
        <>
          <ul className="divide-y divide-border border-y border-border">
            {data.expenses.map((expense) => (
              <ExpenseRow key={expense.id} expense={expense} />
            ))}
          </ul>

          {data.nextOffset !== null ? (
            <div
              ref={sentinelRef}
              className="flex flex-col items-center gap-2 py-4"
            >
              {loadMoreError ? (
                <>
                  <p role="alert" className="text-sm text-destructive">
                    {loadMoreError}
                  </p>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => void loadMore()}
                  >
                    Retry
                  </Button>
                </>
              ) : (
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={loadingMore}
                  onClick={() => void loadMore()}
                >
                  {loadingMore ? (
                    <>
                      <Spinner className="size-4" />
                      Loading...
                    </>
                  ) : (
                    "Load more"
                  )}
                </Button>
              )}
            </div>
          ) : (
            <p className="py-4 text-center text-xs text-muted-foreground">
              All expenses loaded.
            </p>
          )}
        </>
      )}
    </main>
  )
}

async function loadShared(
  link: ShareLink,
  projectId: string
): Promise<SharedData> {
  const category =
    projectId !== "" ? await getCategory(link.orgId, projectId) : null

  if (link.resourceType === "expense") {
    const expense = await getExpense(link.orgId, link.resourceId)
    return {
      category,
      expenses: [expense],
      summary: [totalsFromExpense(expense)],
      nextOffset: null,
    }
  }

  const [list, summary] = await Promise.all([
    listExpenses(link.orgId, projectId || undefined, {
      limit: SHARED_PAGE_SIZE,
    }),
    summarizeExpenses(link.orgId, projectId || undefined),
  ])
  return {
    category,
    expenses: list.items,
    summary: summarizeByCurrency(summary.items),
    nextOffset: list.nextOffset,
  }
}

function Summary({ summary }: { summary: CurrencyTotals[] }) {
  return (
    <section
      aria-label="Shared expense totals"
      className={cn("grid gap-3", summary.length > 1 && "sm:grid-cols-2")}
    >
      {summary.map((item) => {
        const expenseLabel = item.totalCount === 1 ? "expense" : "expenses"
        const categoryLabel =
          item.projectCount === 1 ? "category" : "categories"
        const meta =
          item.projectCount > 1
            ? `${item.totalCount} ${expenseLabel} · ${item.projectCount} ${categoryLabel}`
            : `${item.totalCount} ${expenseLabel}`

        return (
          <Card key={item.currency}>
            <CardContent className="flex flex-col gap-3 py-4">
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <span className="text-[10px] font-semibold tracking-[0.16em] text-muted-foreground uppercase">
                    {item.currency}
                  </span>
                  <span
                    className={cn(
                      "mt-1 block truncate font-display text-3xl leading-none font-semibold tabular-nums",
                      item.netTotal < 0
                        ? "text-spend"
                        : item.netTotal > 0
                          ? "text-credit"
                          : "text-foreground"
                    )}
                  >
                    {formatMoney(item.netTotal, item.currency)}
                  </span>
                  <span className="mt-1 block text-[10px] font-semibold tracking-[0.12em] text-muted-foreground uppercase">
                    Net balance
                  </span>
                </div>
                <span className="shrink-0 rounded-full bg-muted px-2.5 py-1 text-[10px] font-semibold tracking-[0.1em] text-muted-foreground uppercase">
                  {meta}
                </span>
              </div>

              <div className="grid grid-cols-2 divide-x divide-border border-t border-border pt-3">
                <SummaryStat
                  label="Credit"
                  hint="Money in"
                  value={formatMoney(item.totalCredit, item.currency)}
                  dotClassName="bg-credit"
                />
                <SummaryStat
                  label="Debit"
                  hint="Money out"
                  value={formatMoney(item.totalDebit, item.currency)}
                  dotClassName="bg-spend"
                  className="pl-3"
                />
              </div>
            </CardContent>
          </Card>
        )
      })}
    </section>
  )
}

function SummaryStat({
  label,
  hint,
  value,
  dotClassName,
  className,
}: {
  label: string
  hint: string
  value: string
  dotClassName: string
  className?: string
}) {
  return (
    <div className={cn("min-w-0", className)}>
      <div className="flex items-center gap-1.5 text-[10px] font-semibold tracking-[0.12em] text-muted-foreground uppercase">
        <span
          className={cn("size-1.5 rounded-full", dotClassName)}
          aria-hidden="true"
        />
        {label}
      </div>
      <span
        className="mt-1 block truncate text-lg font-semibold text-foreground tabular-nums"
        title={value}
      >
        {value}
      </span>
      <span className="text-[11px] text-muted-foreground">{hint}</span>
    </div>
  )
}

function ExpenseRow({ expense }: { expense: Expense }) {
  return (
    <li className="flex items-center justify-between gap-4 py-4">
      <div className="min-w-0">
        <p className="truncate text-sm font-medium">
          {expense.description ||
            (expense.type === "debit" ? "Expense" : "Money received")}
        </p>
        <p className="mt-1 text-xs text-muted-foreground">
          {formatExpenseDate(expense.spentAt)}
          {expense.tags.length > 0 ? ` · ${expense.tags.join(", ")}` : ""}
        </p>
      </div>
      <span
        className={
          expense.type === "debit"
            ? "shrink-0 text-sm font-semibold text-spend tabular-nums"
            : "shrink-0 text-sm font-semibold text-credit tabular-nums"
        }
      >
        {expense.type === "debit" ? "−" : "+"}
        {formatMoney(expense.amount, expense.currency)}
      </span>
    </li>
  )
}

function Unavailable({ message }: { message: string }) {
  return (
    <main className="flex min-h-svh items-center justify-center px-4">
      <Card className="max-w-md">
        <CardContent className="flex flex-col items-center gap-3 py-10 text-center">
          <GlobeIcon
            className="size-6 text-muted-foreground"
            aria-hidden="true"
          />
          <p className="font-display text-2xl tracking-tight">
            Link unavailable
          </p>
          <p className="text-sm text-muted-foreground">{message}</p>
        </CardContent>
      </Card>
    </main>
  )
}
