import { useEffect, useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { GlobeIcon, ReceiptTextIcon } from "lucide-react"

import { Card, CardContent } from "@/components/ui/card"
import { Spinner } from "@/components/ui/spinner"
import {
  getExpense,
  listExpenses,
  summarizeExpenses,
} from "@/features/expenses/api"
import { formatDirectionalMoney } from "@/features/expenses/money"
import type { Expense, ExpenseSummaryItem } from "@/features/expenses/types"
import { getOrg } from "@/features/orgs/api"
import { getProject } from "@/features/projects/api"
import {
  acquireShareLink,
  clearShareToken,
  setShareToken,
  type ShareLink,
} from "@/lib/share-token"

export function SharedPage() {
  const [link] = useState<ShareLink | null>(acquireShareLink)

  useEffect(() => {
    if (link) {
      setShareToken(link.token)
    }
    return () => clearShareToken()
  }, [link])

  if (!link) {
    return <Unavailable message="This share link is missing its token." />
  }

  return <SharedView link={link} />
}

function SharedView({ link }: { link: ShareLink }) {
  const orgQuery = useQuery({
    queryKey: ["shared-org", link.orgId],
    queryFn: () => getOrg(link.orgId),
    enabled: link.orgId !== "",
  })
  const projectId =
    link.projectId ?? (link.resourceType === "project" ? link.resourceId : "")
  const projectQuery = useQuery({
    queryKey: ["shared-project", link.orgId, projectId],
    queryFn: () => getProject(link.orgId, projectId),
    enabled: projectId !== "",
  })
  const expenseQuery = useQuery({
    queryKey: ["shared-expense", link.orgId, link.resourceId],
    queryFn: () => getExpense(link.orgId, link.resourceId),
    enabled: link.resourceType === "expense" && link.resourceId !== "",
  })
  const listQuery = useQuery({
    queryKey: ["shared-expenses", link.orgId, projectId],
    queryFn: () =>
      listExpenses({
        orgId: link.orgId,
        projectId: projectId || undefined,
        tags: [],
        limit: 20,
        offset: 0,
      }),
    enabled: link.resourceType !== "expense",
  })
  const summaryQuery = useQuery({
    queryKey: ["shared-summary", link.orgId, projectId],
    queryFn: () =>
      summarizeExpenses({
        orgId: link.orgId,
        projectId: projectId || undefined,
        tags: [],
      }),
    enabled: link.resourceType !== "expense",
  })

  const failure =
    orgQuery.error ??
    projectQuery.error ??
    expenseQuery.error ??
    listQuery.error ??
    summaryQuery.error
  if (failure) {
    return <Unavailable message="This link is no longer available." />
  }

  const orgName = orgQuery.data?.name ?? "Shared view"
  const projectName = projectQuery.data?.name ?? "Project"
  const summary = summaryQuery.data?.items ?? []
  const expenses = expenseQuery.data
    ? [expenseQuery.data]
    : (listQuery.data?.items ?? [])
  const loading =
    orgQuery.isPending ||
    (link.resourceType === "expense" ? expenseQuery.isPending : false) ||
    (link.resourceType !== "org" ? projectQuery.isPending : false) ||
    (link.resourceType !== "expense"
      ? listQuery.isPending || summaryQuery.isPending
      : false)

  return (
    <main className="mx-auto flex w-full max-w-4xl flex-col gap-6 px-4 py-8 sm:py-12">
      <header className="flex flex-col gap-2 border-b border-border pb-5">
        <span className="flex items-center gap-2 text-label text-primary uppercase">
          <GlobeIcon className="size-3.5" aria-hidden="true" />
          Shared read-only view
        </span>
        <h1 className="font-display text-heading text-balance">
          {link.resourceType === "org" ? orgName : projectName}
        </h1>
        <p className="text-sm text-muted-foreground">
          {link.resourceType === "org"
            ? `Expenses shared by ${orgName}.`
            : `Read-only expenses from ${orgName}.`}
        </p>
      </header>

      {loading ? (
        <div className="flex justify-center py-10">
          <Spinner className="size-5 text-muted-foreground" />
        </div>
      ) : (
        <>
          {link.resourceType !== "expense" && summary.length > 0 ? (
            <Summary summary={summary} />
          ) : null}

          {expenses.length === 0 ? (
            <Card>
              <CardContent className="flex flex-col items-center gap-2 py-12 text-center text-muted-foreground">
                <ReceiptTextIcon className="size-5" aria-hidden="true" />
                <p className="text-sm">No expenses to show.</p>
              </CardContent>
            </Card>
          ) : (
            <ul className="divide-y divide-border border-y border-border">
              {expenses.map((expense) => (
                <ExpenseRow key={expense.id} expense={expense} />
              ))}
            </ul>
          )}
        </>
      )}
    </main>
  )
}

function Summary({ summary }: { summary: ExpenseSummaryItem[] }) {
  return (
    <div className="grid gap-3 sm:grid-cols-3">
      {summary.slice(0, 3).map((item) => (
        <Card key={`${item.projectId}-${item.currency}`}>
          <CardContent className="flex flex-col gap-1 py-4">
            <span className="text-label text-muted-foreground uppercase">
              {item.currency}
            </span>
            <span className="font-display text-xl font-semibold tabular-nums">
              {formatDirectionalMoney(
                Math.abs(item.netTotal),
                item.currency,
                item.netTotal < 0 ? "debit" : "credit"
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
  const label = expense.description.trim() || "Untitled expense"
  return (
    <li className="flex items-center justify-between gap-4 py-4">
      <div className="flex min-w-0 flex-col gap-1">
        <span className="truncate font-display text-lg font-semibold tracking-tight">
          {label}
        </span>
        <span className="text-xs text-muted-foreground">
          {new Date(expense.spentAt).toLocaleDateString(undefined, {
            year: "numeric",
            month: "short",
            day: "numeric",
          })}
          {expense.tags.length > 0 ? ` · ${expense.tags.join(", ")}` : ""}
        </span>
      </div>
      <span className="shrink-0 font-display text-xl font-semibold tabular-nums">
        {formatDirectionalMoney(expense.amount, expense.currency, expense.type)}
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
          <p className="font-display text-2xl font-semibold tracking-tight">
            Link unavailable
          </p>
          <p className="text-sm text-muted-foreground">{message}</p>
        </CardContent>
      </Card>
    </main>
  )
}
