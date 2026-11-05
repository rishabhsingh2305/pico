import { Link } from "react-router"

import { SectionHeading } from "@/features/dashboard/components/section-heading"
import {
  flowWidth,
  maxFlowValue,
  type CurrencySummary,
} from "@/features/dashboard/data"
import { formatMoney } from "@/features/expenses/money"
import { cn } from "cn"

type ProjectFlowProps = {
  summary: CurrencySummary
  orgId: string
}

export function ProjectFlow({ summary, orgId }: ProjectFlowProps) {
  const max = maxFlowValue(summary.projects)
  const projectLabel = summary.projects.length === 1 ? "project" : "projects"

  return (
    <section
      aria-labelledby="project-flow-heading"
      className="flex flex-col gap-4"
    >
      <SectionHeading
        id="project-flow-heading"
        index="02"
        eyebrow="Flow"
        title="Project net flow"
        meta={`${summary.currency} · ${summary.projects.length} ${projectLabel}`}
      />

      <ul className="divide-y divide-border border-y border-border">
        {summary.projects.map(({ project, totals }, index) => {
          const debitWidth = flowWidth(totals.totalDebit, max)
          const creditWidth = flowWidth(totals.totalCredit, max)
          const net = totals.netTotal
          const entryLabel = totals.totalCount === 1 ? "entry" : "entries"

          return (
            <li key={project.id}>
              <Link
                to={`/orgs/${orgId}/projects/${project.id}`}
                className="group grid gap-3 px-1 py-4 focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none sm:grid-cols-[2rem_minmax(0,1fr)_minmax(10rem,1.1fr)_auto] sm:items-center sm:gap-5"
              >
                <span
                  className="font-mono text-[10px] text-muted-foreground tabular-nums"
                  aria-hidden="true"
                >
                  {String(index + 1).padStart(2, "0")}
                </span>

                <div className="min-w-0">
                  <p className="truncate font-display text-lg font-semibold tracking-tight group-hover:underline">
                    {project.name}
                  </p>
                  <p className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 font-mono text-[10px] tracking-[0.12em] text-muted-foreground uppercase">
                    <span>
                      {totals.totalCount} {entryLabel}
                    </span>
                    {project.tags.length > 0 ? (
                      <span className="text-muted-foreground/70">
                        {project.tags.join(" / ")}
                      </span>
                    ) : null}
                  </p>
                </div>

                <div>
                  <div className="flex items-center" aria-hidden="true">
                    <div className="flex h-2.5 flex-1 justify-end bg-border/40">
                      <div
                        className="h-full bg-primary"
                        style={{ width: `${debitWidth}%` }}
                      />
                    </div>
                    <span className="h-5 w-px bg-foreground/40" />
                    <div className="h-2.5 flex-1 bg-border/40">
                      <div
                        className="h-full bg-chart-2"
                        style={{ width: `${creditWidth}%` }}
                      />
                    </div>
                  </div>
                  <p className="mt-1.5 flex flex-wrap justify-between gap-2 font-mono text-[10px] tracking-[0.08em] uppercase">
                    <span className="text-primary">
                      {formatMoney(totals.totalDebit, summary.currency)} debit
                    </span>
                    <span className="text-chart-2">
                      {formatMoney(totals.totalCredit, summary.currency)} credit
                    </span>
                  </p>
                </div>

                <p
                  className={cn(
                    "font-display text-2xl font-semibold tracking-tight tabular-nums sm:text-right",
                    net < 0
                      ? "text-primary"
                      : net > 0
                        ? "text-chart-2"
                        : "text-muted-foreground"
                  )}
                >
                  {formatMoney(net, summary.currency)}
                </p>
              </Link>
            </li>
          )
        })}
      </ul>

      <div className="flex flex-wrap items-center gap-4 font-mono text-[10px] tracking-[0.12em] text-muted-foreground uppercase">
        <span className="flex items-center gap-1.5">
          <span className="size-2 bg-primary" aria-hidden="true" />
          Debit
        </span>
        <span className="flex items-center gap-1.5">
          <span className="size-2 bg-chart-2" aria-hidden="true" />
          Credit
        </span>
        <span className="ml-auto">Bars scale to the largest value</span>
      </div>
    </section>
  )
}
