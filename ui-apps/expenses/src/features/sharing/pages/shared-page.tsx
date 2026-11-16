import { useEffect, useState } from "react"
import { GlobeIcon } from "lucide-react"

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
import type { Expense, ExpenseSummaryItem } from "@/features/expenses/types"
import {
  acquireShareLink,
  clearShareToken,
  setShareToken,
  type ShareLink,
} from "@/lib/share-token"

type SharedData = {
  category: Category | null
  expenses: Expense[]
  summary: ExpenseSummaryItem[]
}

type SharedStatus = "loading" | "ready" | "error"

export function SharedPage() {
  const [link] = useState<ShareLink | null>(acquireShareLink)
  const [data, setData] = useState<SharedData | null>(null)
  const [status, setStatus] = useState<SharedStatus>("loading")

  useEffect(() => {
    if (!link) {
      return
    }
    setShareToken(link.token)
    let cancelled = false
    loadShared(link)
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
  }, [link])

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

  const currency = data.category?.currency ?? "INR"
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

      {data.summary.length > 0 ? (
        <Summary summary={data.summary} fallbackCurrency={currency} />
      ) : null}

      {data.expenses.length === 0 ? (
        <Card>
          <CardContent className="py-10 text-center text-sm text-muted-foreground">
            No expenses to show.
          </CardContent>
        </Card>
      ) : (
        <ul className="divide-y divide-border border-y border-border">
          {data.expenses.map((expense) => (
            <ExpenseRow key={expense.id} expense={expense} />
          ))}
        </ul>
      )}
    </main>
  )
}

async function loadShared(link: ShareLink): Promise<SharedData> {
  const projectId =
    link.projectId ?? (link.resourceType === "project" ? link.resourceId : "")
  const category =
    projectId !== "" ? await getCategory(link.orgId, projectId) : null

  if (link.resourceType === "expense") {
    const expense = await getExpense(link.orgId, link.resourceId)
    return { category, expenses: [expense], summary: [] }
  }

  const [list, summary] = await Promise.all([
    listExpenses(link.orgId, projectId || undefined, { limit: 100 }),
    summarizeExpenses(link.orgId, projectId || undefined),
  ])
  return { category, expenses: list.items, summary: summary.items }
}

function Summary({
  summary,
  fallbackCurrency,
}: {
  summary: ExpenseSummaryItem[]
  fallbackCurrency: string
}) {
  return (
    <div className="grid gap-3 sm:grid-cols-3">
      {summary.slice(0, 3).map((item) => (
        <Card key={`${item.projectId}-${item.currency}`}>
          <CardContent className="flex flex-col gap-1 py-4">
            <span className="text-[10px] font-semibold tracking-[0.16em] text-muted-foreground uppercase">
              {item.currency}
            </span>
            <span className="font-display text-xl font-semibold tabular-nums">
              {formatMoney(
                Math.abs(item.netTotal),
                item.currency || fallbackCurrency
              )}
            </span>
            <span className="text-xs text-muted-foreground">
              {item.totalCount} {item.totalCount === 1 ? "expense" : "expenses"}
            </span>
          </CardContent>
        </Card>
      ))}
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
