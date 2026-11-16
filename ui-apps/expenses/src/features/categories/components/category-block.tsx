import { Link } from "react-router"
import { ArrowUpRightIcon, Link2Icon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { categoryColor } from "@/features/categories/colors"
import { CategoryGlyph } from "@/features/categories/icons"
import { PublicShareBadge } from "@/features/sharing/components/public-share-badge"
import type { Category } from "@/features/categories/types"
import { formatCompactMoney, formatMoney } from "@/features/expenses/format"

type CategoryBlockProps = {
  category: Category
  monthlyTotal?: number | null
  entryCount?: number
  share?: number
  shared?: boolean
  canShare?: boolean
  onShare?: () => void
}

export function CategoryBlock({
  category,
  monthlyTotal,
  entryCount = 0,
  share = 0,
  shared = false,
  canShare = false,
  onShare,
}: CategoryBlockProps) {
  const block = categoryColor(category.id)
  const total = monthlyTotal ?? 0

  return (
    <div className="group -mx-2 flex items-center gap-2 rounded-lg px-2 py-3.5 transition-colors hover:bg-muted/50">
      <Link
        to={`/categories/${category.id}`}
        className="flex min-w-0 flex-1 items-center gap-3 rounded-sm focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background focus-visible:outline-none"
      >
        <span
          className="flex size-10 shrink-0 items-center justify-center rounded-lg shadow-[inset_0_1px_0_rgba(255,255,255,0.7),0_1px_2px_rgba(16,24,40,0.08)]"
          style={{
            backgroundColor: block.background,
            color: block.foreground,
          }}
        >
          <CategoryGlyph
            name={category.name}
            id={category.id}
            className="size-[18px]"
          />
        </span>

        <div className="min-w-0 flex-1">
          <div className="flex items-baseline justify-between gap-3">
            <div className="min-w-0">
              <span className="block truncate text-sm font-medium text-foreground">
                {category.name}
              </span>
              <span className="mt-0.5 block text-[10px] text-muted-foreground">
                {entryCount} {entryCount === 1 ? "entry" : "entries"}
              </span>
            </div>
            <span
              className="shrink-0 text-base font-semibold tracking-tight text-foreground tabular-nums"
              title={formatMoney(total, category.currency)}
              aria-label={formatMoney(total, category.currency)}
            >
              {formatCompactMoney(total, category.currency)}
            </span>
          </div>
          <div className="mt-2 h-1 overflow-hidden rounded-full bg-muted">
            <div
              className="h-full rounded-full transition-[width] duration-500"
              style={{
                width: `${share > 0 ? Math.max(2, Math.min(100, share)) : 0}%`,
                backgroundColor: block.foreground,
              }}
            />
          </div>
        </div>

        <ArrowUpRightIcon
          className="size-4 shrink-0 text-muted-foreground transition-all group-hover:translate-x-0.5 group-hover:-translate-y-0.5 group-hover:text-foreground"
          aria-hidden="true"
        />
      </Link>
      {shared ? <PublicShareBadge /> : null}
      {canShare ? (
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          aria-label={`Share ${category.name}`}
          onClick={onShare}
        >
          <Link2Icon aria-hidden="true" />
        </Button>
      ) : null}
    </div>
  )
}
