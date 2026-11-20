import { Button } from "@/components/ui/button"

type ExpensePaginationProps = {
  showingStart: number
  showingEnd: number
  totalCount: number
  offset: number
  pageSize: number
  nextOffset: number | null
  onChange: (offset: number) => void
  label?: string
}

export function ExpensePagination({
  showingStart,
  showingEnd,
  totalCount,
  offset,
  pageSize,
  nextOffset,
  onChange,
  label = "Expense pagination",
}: ExpensePaginationProps) {
  return (
    <nav
      aria-label={label}
      className="flex flex-wrap items-center justify-between gap-3"
    >
      <p
        className="font-mono text-[11px] tracking-[0.14em] text-muted-foreground uppercase"
        aria-live="polite"
      >
        Showing {showingStart}–{showingEnd} of {totalCount}
      </p>
      <div className="flex gap-2">
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={offset === 0}
          onClick={() => onChange(Math.max(0, offset - pageSize))}
        >
          Previous
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={nextOffset === null}
          onClick={() => {
            if (nextOffset !== null) {
              onChange(nextOffset)
            }
          }}
        >
          Next
        </Button>
      </div>
    </nav>
  )
}
