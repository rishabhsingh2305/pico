import { CountUp } from "@/components/count-up"
import type { CurrencySummary } from "@/features/dashboard/data"
import { formatMoney } from "@/features/expenses/money"
import { cn } from "cn"

type DashboardSummaryProps = {
  summary: CurrencySummary
  currencies: string[]
  selectedCurrency: string
  onCurrencyChange: (currency: string) => void
}

export function DashboardSummary({
  summary,
  currencies,
  selectedCurrency,
  onCurrencyChange,
}: DashboardSummaryProps) {
  const projectLabel = summary.projects.length === 1 ? "project" : "projects"
  const entryLabel = summary.totalCount === 1 ? "entry" : "entries"
  const summaryText = `${summary.totalCount} ${entryLabel} across ${
    summary.projects.length
  } ${projectLabel}. Net ${formatMoney(
    summary.netTotal,
    summary.currency
  )}. Credit ${formatMoney(
    summary.totalCredit,
    summary.currency
  )}. Debit ${formatMoney(summary.totalDebit, summary.currency)}.`

  return (
    <section
      aria-label="Workspace totals"
      className="border border-border bg-card"
    >
      <div className="relative p-6 sm:p-8 lg:p-10">
        <span
          className={cn(
            "absolute inset-x-0 top-0 h-1",
            summary.netTotal < 0 ? "bg-primary" : "bg-chart-2"
          )}
          aria-hidden="true"
        />

        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <p className="text-label text-muted-foreground uppercase">
              Net across {summary.projects.length} {projectLabel}
            </p>
            <p className="mt-1 text-sm text-muted-foreground">
              Credits minus debits, in {summary.currency}.
            </p>
          </div>

          {currencies.length > 1 ? (
            <div
              className="flex flex-wrap gap-1"
              role="group"
              aria-label="Currency"
            >
              {currencies.map((currency) => (
                <button
                  key={currency}
                  type="button"
                  aria-pressed={currency === selectedCurrency}
                  onClick={() => onCurrencyChange(currency)}
                  className={cn(
                    "border px-2.5 py-1 font-mono text-[10px] tracking-[0.14em] uppercase transition-colors focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none",
                    currency === selectedCurrency
                      ? "border-foreground bg-foreground text-background"
                      : "border-border bg-background text-muted-foreground hover:border-foreground/40 hover:text-foreground"
                  )}
                >
                  {currency}
                </button>
              ))}
            </div>
          ) : (
            <span className="font-mono text-[10px] tracking-[0.14em] text-muted-foreground uppercase">
              {summary.currency}
            </span>
          )}
        </div>

        <span
          aria-hidden="true"
          className={cn(summary.netTotal < 0 && "text-primary")}
        >
          <CountUp
            value={summary.netTotal}
            format={(amount) => formatMoney(amount, summary.currency)}
            className="mt-6 block max-w-full font-display text-metric whitespace-nowrap tabular-nums"
            durationMs={800}
          />
        </span>

        <p
          className="mt-4 font-mono text-[11px] tracking-[0.16em] text-muted-foreground uppercase"
          aria-hidden="true"
        >
          {summary.totalCount} {entryLabel} · {summary.currency}
        </p>
        <p className="sr-only" aria-live="polite">
          {summaryText}
        </p>
      </div>

      <dl className="grid border-t border-border sm:grid-cols-3">
        <Stat
          label="Credit"
          value={formatMoney(summary.totalCredit, summary.currency)}
          className="bg-chart-2/10 text-chart-2"
        />
        <Stat
          label="Debit"
          value={formatMoney(summary.totalDebit, summary.currency)}
          className="bg-primary/10 text-primary"
        />
        <Stat
          label="Projects"
          value={String(summary.projects.length)}
          className="bg-accent text-accent-foreground"
        />
      </dl>
    </section>
  )
}

function Stat({
  label,
  value,
  className,
}: {
  label: string
  value: string
  className?: string
}) {
  return (
    <div
      className={cn(
        "flex flex-col gap-1 border-t border-border px-6 py-5 first:border-t-0 sm:border-t-0 sm:border-l sm:first:border-l-0",
        className
      )}
    >
      <dt className="text-label uppercase opacity-70">{label}</dt>
      <dd className="font-display text-2xl font-semibold tracking-tight tabular-nums sm:text-3xl">
        {value}
      </dd>
    </div>
  )
}
