import { Link } from "react-router"
import { ArrowUpRightIcon } from "lucide-react"

import { categoryColor } from "@/features/categories/colors"
import type { Category } from "@/features/categories/types"
import { formatCompactMoney, formatMoney } from "@/features/expenses/format"
import { cn } from "@/lib/utils"

type CategoryBlockProps = {
  category: Category
  featured?: boolean
  monthlyTotal?: number | null
}

export function CategoryBlock({
  category,
  featured = false,
  monthlyTotal,
}: CategoryBlockProps) {
  const block = categoryColor(category.id)

  return (
    <Link
      to={`/categories/${category.id}`}
      className={cn(
        "group flex flex-col justify-between border-2 border-foreground p-4 transition-transform hover:-translate-y-1 focus-visible:ring-2 focus-visible:ring-foreground focus-visible:ring-offset-2 focus-visible:ring-offset-background focus-visible:outline-none motion-reduce:transform-none sm:p-5",
        featured ? "min-h-44 sm:col-span-2 sm:min-h-56" : "min-h-32 sm:min-h-40"
      )}
      style={{ backgroundColor: block.background, color: block.foreground }}
    >
      <span
        className={cn(
          "font-display leading-[0.95] tracking-tight",
          featured ? "text-3xl sm:text-5xl" : "text-2xl sm:text-4xl"
        )}
      >
        {category.name}
      </span>

      <div className="mt-3 flex items-end justify-between gap-2">
        <div className="flex flex-col gap-0.5">
          <span className="text-[0.65rem] font-semibold opacity-70 sm:text-xs">
            Spent this month
          </span>
          <span
            className={cn(
              "font-display leading-none",
              featured ? "text-2xl sm:text-3xl" : "text-lg sm:text-2xl"
            )}
            title={
              monthlyTotal === undefined || monthlyTotal === null
                ? undefined
                : formatMoney(monthlyTotal, category.currency)
            }
            aria-label={
              monthlyTotal === undefined || monthlyTotal === null
                ? undefined
                : formatMoney(monthlyTotal, category.currency)
            }
          >
            {monthlyTotal === undefined || monthlyTotal === null
              ? "—"
              : formatCompactMoney(monthlyTotal, category.currency)}
          </span>
        </div>
        <ArrowUpRightIcon
          className="size-5 shrink-0 transition-transform group-hover:translate-x-0.5 group-hover:-translate-y-0.5"
          aria-hidden="true"
        />
      </div>
    </Link>
  )
}
