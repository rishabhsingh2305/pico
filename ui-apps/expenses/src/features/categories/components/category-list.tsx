import { useCallback, useEffect, useState } from "react"
import { CircleAlertIcon, PlusIcon } from "lucide-react"

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
import { useExpenses } from "@/features/expenses/expenses-provider"
import { monthRange } from "@/features/expenses/months"
import { ApiError } from "@/lib/api-client"

type Status = "loading" | "ready" | "error"

export function CategoryList() {
  const { orgId } = useExpenses()
  const { logout } = useAuth()
  const [categories, setCategories] = useState<Category[]>([])
  const [status, setStatus] = useState<Status>("loading")
  const [error, setError] = useState("")
  const [isFormOpen, setIsFormOpen] = useState(false)
  const [attempt, setAttempt] = useState(0)
  const [monthlyTotals, setMonthlyTotals] = useState<
    Record<string, number | null>
  >({})
  const monthLabel = currentMonthLabel()

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

    summarizeExpenses(orgId, undefined, {
      from: range.from,
      to: range.to,
    })
      .then((response) => {
        if (!cancelled) {
          const totals = Object.fromEntries(
            response.items.map((item) => [item.projectId, item.totalDebit])
          )
          setMonthlyTotals(totals)
        }
      })
      .catch(() => {
        if (!cancelled) {
          setMonthlyTotals({})
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

  return (
    <div className="flex flex-col gap-5">
      <section className="border-2 border-foreground bg-primary p-5 text-primary-foreground sm:p-8">
        <div className="flex flex-col gap-5 sm:flex-row sm:items-end sm:justify-between sm:gap-6">
          <div>
            <h1 className="font-display text-4xl leading-[0.9] tracking-tight sm:text-7xl">
              Your spending
            </h1>
            <p className="mt-3 max-w-sm text-sm font-medium text-primary-foreground/80 sm:mt-4">
              Group your expenses into simple categories.
            </p>
          </div>
          <Button
            type="button"
            className="h-12 border-foreground bg-accent text-accent-foreground hover:bg-foreground hover:text-background sm:h-11"
            onClick={() => setIsFormOpen(true)}
            disabled={status !== "ready"}
          >
            <PlusIcon data-icon="inline-start" aria-hidden="true" />
            New category
          </Button>
        </div>

        <div className="mt-6 flex items-center justify-between border-t-2 border-primary-foreground/20 pt-3 text-sm font-semibold text-primary-foreground/80 sm:mt-8 sm:pt-4">
          <span>
            {status === "ready"
              ? `${categories.length} ${categories.length === 1 ? "category" : "categories"}`
              : "—"}
          </span>
          <span>{monthLabel}</span>
        </div>
      </section>

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
        <div className="grid gap-3 sm:grid-cols-2 sm:gap-4">
          {categories.map((category, index) => (
            <CategoryBlock
              key={category.id}
              category={category}
              featured={index === 0}
              monthlyTotal={monthlyTotals[category.id]}
            />
          ))}
        </div>
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
    <div className="grid gap-3 sm:grid-cols-2 sm:gap-4">
      <Skeleton className="h-44 border-2 border-foreground bg-muted sm:col-span-2 sm:h-56" />
      <Skeleton className="h-32 border-2 border-foreground bg-muted sm:h-40" />
      <Skeleton className="h-32 border-2 border-foreground bg-muted sm:h-40" />
    </div>
  )
}

function EmptyCategories() {
  return (
    <div className="flex flex-col items-center justify-center border-2 border-foreground bg-card px-6 py-12 text-center sm:py-16">
      <span className="font-display text-8xl leading-none" aria-hidden="true">
        0
      </span>
      <div className="mt-4 h-2 w-16 bg-primary" aria-hidden="true" />
      <h2 className="mt-4 font-display text-2xl tracking-tight">
        No categories yet
      </h2>
      <p className="mt-1 text-sm text-muted-foreground">
        Start with Food, Travel, or Bills.
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
