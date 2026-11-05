import { Link } from "react-router"
import { ArrowUpRightIcon } from "lucide-react"

import { SectionHeading } from "@/features/dashboard/components/section-heading"
import type { ProjectTotals } from "@/features/dashboard/data"
import { formatDirectionalMoney } from "@/features/expenses/money"
import type { Expense } from "@/features/expenses/types"
import { cn } from "cn"

type RecentActivityProps = {
  expenses: Expense[]
  projects: ProjectTotals[]
  orgId: string
}

export function RecentActivity({
  expenses,
  projects,
  orgId,
}: RecentActivityProps) {
  const projectNames = new Map(
    projects.map(({ project }) => [project.id, project.name])
  )

  return (
    <section
      aria-labelledby="recent-activity-heading"
      className="flex flex-col gap-4"
    >
      <SectionHeading
        id="recent-activity-heading"
        index="03"
        eyebrow="Activity"
        title="Recent movement"
        meta="Latest entries"
      />

      {expenses.length === 0 ? (
        <div className="flex flex-col items-start gap-2 border-y border-border py-10">
          <p className="font-display text-2xl font-semibold tracking-tight">
            No activity yet
          </p>
          <p className="text-sm text-muted-foreground">
            Expenses will appear here as soon as they are recorded.
          </p>
        </div>
      ) : (
        <ul className="divide-y divide-border border-y border-border">
          {expenses.map((expense) => {
            const isCredit = expense.type === "credit"
            const description = expense.description.trim() || "Untitled expense"
            const projectName =
              projectNames.get(expense.projectId) ?? "Unknown project"

            return (
              <li
                key={expense.id}
                className="flex flex-col gap-2 py-4 sm:flex-row sm:items-center sm:justify-between sm:gap-4"
              >
                <div className="min-w-0">
                  <p className="truncate font-display text-lg font-semibold tracking-tight">
                    {description}
                  </p>
                  <p className="mt-1 font-mono text-[10px] tracking-[0.12em] text-muted-foreground uppercase">
                    {formatActivityDate(expense.spentAt)} · {projectName}
                  </p>
                </div>

                <div className="flex shrink-0 items-center justify-between gap-3 sm:justify-end">
                  <span
                    className={cn(
                      "text-label uppercase",
                      isCredit ? "text-chart-2" : "text-primary"
                    )}
                  >
                    {isCredit ? "Credit" : "Debit"}
                  </span>
                  <span
                    className={cn(
                      "font-display text-xl font-semibold tracking-tight tabular-nums",
                      isCredit && "text-chart-2"
                    )}
                  >
                    {formatDirectionalMoney(
                      expense.amount,
                      expense.currency,
                      expense.type
                    )}
                  </span>
                  <span className="font-mono text-[10px] tracking-[0.12em] text-muted-foreground">
                    {expense.currency}
                  </span>
                </div>
              </li>
            )
          })}
        </ul>
      )}

      <Link
        to={`/orgs/${orgId}`}
        className="inline-flex w-fit items-center gap-1.5 text-label text-primary uppercase underline-offset-4 hover:underline focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
      >
        Open organization
        <ArrowUpRightIcon className="size-3.5" aria-hidden="true" />
      </Link>
    </section>
  )
}

function formatActivityDate(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
  }).format(date)
}
