import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type PointerEvent as ReactPointerEvent,
} from "react"
import { CircleAlertIcon, PlusIcon, WifiIcon } from "lucide-react"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { useAuth } from "@/features/auth/auth-provider"
import { listCategories } from "@/features/categories/api"
import { CategoryBlock } from "@/features/categories/components/category-block"
import { CategoryForm } from "@/features/categories/components/category-form"
import { categoryErrorMessage } from "@/features/categories/error-message"
import type { Category } from "@/features/categories/types"
import { summarizeExpenses } from "@/features/expenses/api"
import {
  formatCompactMoney,
  formatReadableMoney,
} from "@/features/expenses/format"
import { useExpenses } from "@/features/expenses/expenses-provider"
import { monthRange, previousMonthRange } from "@/features/expenses/months"
import type { ExpenseSummaryItem } from "@/features/expenses/types"
import { useDeviceTilt } from "@/hooks/use-device-tilt"
import { ApiError } from "@/lib/api-client"
import { cn } from "@/lib/utils"

type Status = "loading" | "ready" | "error"

export function CategoryList() {
  const { orgId } = useExpenses()
  const { logout } = useAuth()
  const cardRef = useRef<HTMLElement>(null)
  useDeviceTilt(cardRef)
  const [categories, setCategories] = useState<Category[]>([])
  const [status, setStatus] = useState<Status>("loading")
  const [error, setError] = useState("")
  const [isFormOpen, setIsFormOpen] = useState(false)
  const [attempt, setAttempt] = useState(0)
  const [monthlySummary, setMonthlySummary] = useState<
    Record<string, ExpenseSummaryItem>
  >({})
  const [previousMonthTotal, setPreviousMonthTotal] = useState(0)
  const [summaryStatus, setSummaryStatus] = useState<
    "loading" | "ready" | "error"
  >("loading")
  const monthLabel = currentMonthLabel()
  const currency = categories[0]?.currency ?? "INR"
  const monthlyTotal = Object.values(monthlySummary).reduce(
    (total, item) => total + item.totalDebit,
    0
  )
  const monthlyCredit = Object.values(monthlySummary).reduce(
    (total, item) => total + item.totalCredit,
    0
  )
  const entryCount = Object.values(monthlySummary).reduce(
    (total, item) => total + item.totalCount,
    0
  )
  const daysElapsed = Math.max(1, new Date().getDate())
  const dailyAverage = Math.round(monthlyTotal / daysElapsed)
  const monthChange =
    previousMonthTotal > 0
      ? ((monthlyTotal - previousMonthTotal) / previousMonthTotal) * 100
      : null
  const cashFlowTotal = monthlyTotal + monthlyCredit
  const spentShare =
    cashFlowTotal > 0 ? (monthlyTotal / cashFlowTotal) * 100 : 0
  const receivedShare =
    cashFlowTotal > 0 ? (monthlyCredit / cashFlowTotal) * 100 : 0

  useEffect(() => {
    let cancelled = false

    listCategories(orgId)
      .then((items) => {
        if (cancelled) {
          return
        }
        setCategories(items)
        setStatus("ready")
      })
      .catch(async (caught: unknown) => {
        if (cancelled) {
          return
        }
        if (caught instanceof ApiError && caught.status === 401) {
          await logout()
          return
        }
        setError(categoryErrorMessage(caught))
        setStatus("error")
      })

    return () => {
      cancelled = true
    }
  }, [orgId, logout, attempt])

  const retry = useCallback(() => {
    setStatus("loading")
    setError("")
    setAttempt((current) => current + 1)
  }, [])

  useEffect(() => {
    if (status !== "ready" || categories.length === 0) {
      return
    }

    let cancelled = false
    const range = monthRange()
    const previousRange = previousMonthRange()

    Promise.all([
      summarizeExpenses(orgId, undefined, {
        from: range.from,
        to: range.to,
      }),
      summarizeExpenses(orgId, undefined, {
        from: previousRange.from,
        to: previousRange.to,
      }).catch(() => ({ items: [] })),
    ])
      .then(([response, previousResponse]) => {
        if (!cancelled) {
          setMonthlySummary(
            Object.fromEntries(
              response.items.map((item) => [item.projectId, item])
            )
          )
          setPreviousMonthTotal(
            previousResponse.items.reduce(
              (total, item) => total + item.totalDebit,
              0
            )
          )
          setSummaryStatus("ready")
        }
      })
      .catch(() => {
        if (!cancelled) {
          setMonthlySummary({})
          setPreviousMonthTotal(0)
          setSummaryStatus("error")
        }
      })

    return () => {
      cancelled = true
    }
  }, [orgId, categories, status])

  function handleCreated(category: Category) {
    setCategories((current) => [category, ...current])
    setStatus("ready")
  }

  function handleCardPointerMove(event: ReactPointerEvent<HTMLElement>) {
    if (event.pointerType !== "mouse") {
      return
    }

    const bounds = event.currentTarget.getBoundingClientRect()
    const x = (event.clientX - bounds.left) / bounds.width - 0.5
    const y = (event.clientY - bounds.top) / bounds.height - 0.5

    event.currentTarget.style.setProperty("--tilt-x", `${y * -12}deg`)
    event.currentTarget.style.setProperty("--tilt-y", `${x * -12}deg`)
  }

  function resetCardTilt(event: ReactPointerEvent<HTMLElement>) {
    event.currentTarget.style.removeProperty("--tilt-x")
    event.currentTarget.style.removeProperty("--tilt-y")
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="relative">
        <div
          className="pointer-events-none absolute inset-x-4 top-4 -bottom-2 rounded-3xl bg-[#083b2b]/15"
          aria-hidden="true"
        />
        <div
          className="pointer-events-none absolute inset-x-2 top-2 -bottom-1 rounded-3xl bg-[#083b2b]/25"
          aria-hidden="true"
        />
        <section
          ref={cardRef}
          className="statement-card relative overflow-hidden rounded-3xl bg-[linear-gradient(135deg,#0f7053_0%,#084a37_55%,#13785c_100%)] p-5 text-primary-foreground shadow-[0_1px_0_rgba(255,255,255,0.2)_inset,0_-1px_0_rgba(0,0,0,0.22)_inset,0_10px_20px_rgba(9,58,43,0.18),0_32px_70px_rgba(9,58,43,0.3)] ring-1 ring-white/10 ring-inset sm:p-6"
          onPointerMove={handleCardPointerMove}
          onPointerLeave={resetCardTilt}
          onPointerCancel={resetCardTilt}
        >
          <div
            className="pointer-events-none absolute -top-16 -right-12 size-52 rounded-full bg-primary-foreground/[0.05]"
            aria-hidden="true"
          />
          <div
            className="pointer-events-none absolute top-20 -right-24 size-52 rounded-full bg-primary-foreground/[0.04]"
            aria-hidden="true"
          />
          <div
            className="pointer-events-none absolute -top-24 -left-16 h-64 w-20 rotate-[24deg] bg-white/[0.07] blur-2xl"
            aria-hidden="true"
          />
          <div
            className="pointer-events-none absolute inset-x-0 top-0 h-px bg-white/20"
            aria-hidden="true"
          />

          <div
            className="absolute top-5 right-5 flex items-center gap-2"
            aria-hidden="true"
          >
            <span className="relative h-6 w-8 overflow-hidden rounded-[4px] bg-gradient-to-br from-[#f3d989] via-[#d8b05a] to-[#a97d32] shadow-[inset_0_1px_0_rgba(255,255,255,0.5)]">
              <span className="absolute inset-x-1 top-1.5 h-px bg-black/20" />
              <span className="absolute inset-x-1 top-3 h-px bg-black/15" />
              <span className="absolute inset-x-1 top-[1.1rem] h-px bg-black/10" />
            </span>
            <WifiIcon className="size-4 rotate-90 text-primary-foreground/40" />
          </div>

          <div className="relative">
            <p className="text-[10px] font-semibold tracking-[0.18em] text-primary-foreground/55 uppercase">
              {monthLabel}
            </p>
            <h1 className="mt-2 text-4xl leading-none font-semibold tracking-tight text-primary-foreground tabular-nums sm:text-5xl">
              {status === "ready"
                ? formatReadableMoney(monthlyTotal, currency)
                : "—"}
            </h1>
            <p className="mt-2 text-sm text-primary-foreground/70">
              {status === "ready"
                ? `Spent across ${categories.length} ${
                    categories.length === 1 ? "category" : "categories"
                  } · ${entryCount} ${entryCount === 1 ? "entry" : "entries"}`
                : "Loading your categories"}
            </p>

            <div className="mt-5">
              <div className="flex h-1.5 overflow-hidden rounded-full bg-primary-foreground/15">
                <span
                  className="h-full bg-[#e07a66] transition-[width] duration-500"
                  style={{ width: `${spentShare}%` }}
                />
                <span
                  className="h-full bg-[#63c7a3] transition-[width] duration-500"
                  style={{ width: `${receivedShare}%` }}
                />
              </div>
              <div className="mt-2 flex items-center justify-between gap-3 text-[10px] font-medium tracking-[0.06em] text-primary-foreground/65 uppercase">
                <span className="flex items-center gap-1.5">
                  <span className="size-1.5 rounded-full bg-[#e07a66]" />
                  Spent {formatCompactMoney(monthlyTotal, currency)}
                </span>
                <span className="flex items-center gap-1.5">
                  <span className="size-1.5 rounded-full bg-[#63c7a3]" />
                  Received {formatCompactMoney(monthlyCredit, currency)}
                </span>
              </div>
            </div>

            <Button
              type="button"
              className="mt-5 h-11 w-full rounded-xl bg-accent text-accent-foreground shadow-[0_2px_6px_rgba(0,0,0,0.16),inset_0_1px_0_rgba(255,255,255,0.35)] hover:bg-accent/90 sm:w-auto"
              onClick={() => setIsFormOpen(true)}
              disabled={status !== "ready"}
            >
              <PlusIcon data-icon="inline-start" aria-hidden="true" />
              New category
            </Button>
          </div>
        </section>
      </div>

      {status === "ready" && categories.length > 0 ? (
        <section className="grid grid-cols-2 divide-x divide-border rounded-xl border border-border bg-card/70 shadow-[0_1px_2px_rgba(16,24,40,0.03),0_12px_30px_rgba(16,24,40,0.05)]">
          <div className="min-w-0 px-4 py-3">
            <p className="text-[10px] font-semibold tracking-[0.14em] text-muted-foreground uppercase">
              Daily average
            </p>
            <p className="mt-1 truncate text-sm font-semibold text-foreground tabular-nums">
              {summaryStatus === "ready"
                ? formatReadableMoney(dailyAverage, currency)
                : "—"}
            </p>
            <p className="mt-0.5 text-[11px] text-muted-foreground">
              Across {daysElapsed} {daysElapsed === 1 ? "day" : "days"}
            </p>
          </div>
          <div className="min-w-0 px-4 py-3">
            <p className="text-[10px] font-semibold tracking-[0.14em] text-muted-foreground uppercase">
              vs last month
            </p>
            <p
              className={cn(
                "mt-1 truncate text-sm font-semibold tabular-nums",
                summaryStatus === "error"
                  ? "text-muted-foreground"
                  : monthChange === null
                    ? "text-foreground"
                    : monthChange > 0
                      ? "text-spend"
                      : monthChange < 0
                        ? "text-credit"
                        : "text-foreground"
              )}
            >
              {summaryStatus === "error"
                ? "Unavailable"
                : monthChange === null
                  ? monthlyTotal > 0
                    ? "New this month"
                    : "No change"
                  : monthChange === 0
                    ? "No change"
                    : `${monthChange > 0 ? "↑" : "↓"} ${Math.abs(monthChange).toFixed(0)}%`}
            </p>
            <p className="mt-0.5 truncate text-[11px] text-muted-foreground">
              {summaryStatus === "ready"
                ? `${formatCompactMoney(previousMonthTotal, currency)} last month`
                : "Monthly comparison"}
            </p>
          </div>
        </section>
      ) : null}

      {status === "loading" ? <CategorySkeletons /> : null}

      {status === "error" ? (
        <Alert variant="destructive">
          <CircleAlertIcon aria-hidden="true" />
          <AlertDescription className="flex flex-wrap items-center justify-between gap-3">
            <span>{error}</span>
            <Button type="button" size="sm" variant="outline" onClick={retry}>
              Try again
            </Button>
          </AlertDescription>
        </Alert>
      ) : null}

      {status === "ready" && categories.length === 0 ? (
        <EmptyCategories />
      ) : null}

      {status === "ready" && categories.length > 0 ? (
        <section>
          <div className="flex items-center justify-between pb-2 text-[10px] font-semibold tracking-[0.14em] text-muted-foreground uppercase">
            <span>All categories</span>
            <span>{categories.length}</span>
          </div>
          <ul className="divide-y divide-border border-y border-border">
            {categories.map((category) => {
              const summary = monthlySummary[category.id]
              const total = summary?.totalDebit ?? 0
              return (
                <li key={category.id}>
                  <CategoryBlock
                    category={category}
                    monthlyTotal={total}
                    entryCount={summary?.totalCount ?? 0}
                    share={monthlyTotal > 0 ? (total / monthlyTotal) * 100 : 0}
                  />
                </li>
              )
            })}
          </ul>
        </section>
      ) : null}

      <CategoryForm
        open={isFormOpen}
        onOpenChange={setIsFormOpen}
        onCreated={handleCreated}
      />
    </div>
  )
}

function CategorySkeletons() {
  return (
    <div className="flex flex-col divide-y divide-border border-y border-border">
      {Array.from({ length: 4 }, (_, index) => (
        <div key={index} className="flex items-center gap-3 py-3.5">
          <Skeleton className="size-10 rounded-lg bg-muted" />
          <div className="flex-1">
            <Skeleton className="h-3 w-28 bg-muted" />
            <Skeleton className="mt-2 h-1 w-full bg-muted" />
          </div>
          <Skeleton className="h-4 w-16 bg-muted" />
        </div>
      ))}
    </div>
  )
}

function EmptyCategories() {
  return (
    <div className="flex flex-col items-center justify-center border-y border-border px-6 py-14 text-center">
      <span
        className="flex size-14 items-center justify-center rounded-full bg-muted text-muted-foreground"
        aria-hidden="true"
      >
        ₹
      </span>
      <h2 className="mt-4 font-display text-2xl text-foreground">
        No categories yet
      </h2>
      <p className="mt-1 max-w-xs text-sm text-muted-foreground">
        Start with Food, Travel, or Bills. You can add expenses inside each
        category.
      </p>
    </div>
  )
}

function currentMonthLabel(): string {
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    year: "numeric",
  }).format(new Date())
}
