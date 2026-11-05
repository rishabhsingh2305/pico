import { useState } from "react"
import { ReceiptTextIcon, Trash2Icon } from "lucide-react"

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { Spinner } from "@/components/ui/spinner"
import { formatDirectionalMoney } from "@/features/expenses/money"
import type { Expense } from "@/features/expenses/types"
import { cn } from "cn"

type ExpenseListProps = {
  expenses: Expense[]
  deletingId: string | null
  hasFilters?: boolean
  onDelete: (expense: Expense) => void
}

export function ExpenseList({
  expenses,
  deletingId,
  hasFilters = false,
  onDelete,
}: ExpenseListProps) {
  const [pendingDelete, setPendingDelete] = useState<Expense | null>(null)

  function confirmDelete() {
    if (!pendingDelete) {
      return
    }
    onDelete(pendingDelete)
    setPendingDelete(null)
  }

  if (expenses.length === 0) {
    return (
      <div className="flex flex-col items-center gap-3 border-y border-border py-14 text-center">
        <div className="flex size-12 items-center justify-center rounded-md border border-border bg-card text-muted-foreground">
          <ReceiptTextIcon className="size-5" aria-hidden="true" />
        </div>
        <p className="font-display text-2xl font-semibold tracking-tight">
          {hasFilters ? "No matching expenses" : "No expenses yet"}
        </p>
        <p className="max-w-sm text-sm text-muted-foreground">
          {hasFilters
            ? "Try changing the tags or date range."
            : "Use New expense to add the first debit or credit."}
        </p>
      </div>
    )
  }

  return (
    <>
      <div className="border-y border-border">
        <ul className="divide-y divide-border">
          {expenses.map((expense) => {
            const label = expense.description.trim() || "Untitled expense"
            const isDeleting = deletingId === expense.id
            const isCredit = expense.type === "credit"
            return (
              <li
                key={expense.id}
                className="group/row flex flex-col gap-3 px-1 py-5 sm:flex-row sm:items-center sm:justify-between"
              >
                <div className="flex min-w-0 flex-col gap-2">
                  <div className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
                    <span
                      className={cn(
                        "inline-flex items-center gap-1.5 text-label uppercase",
                        isCredit ? "text-chart-2" : "text-primary"
                      )}
                    >
                      <span
                        className={cn(
                          "size-1.5 rounded-full",
                          isCredit ? "bg-chart-2" : "bg-primary"
                        )}
                        aria-hidden="true"
                      />
                      {isCredit ? "Credit" : "Debit"}
                    </span>
                    <p className="min-w-0 truncate font-display text-xl font-semibold tracking-tight">
                      {label}
                    </p>
                  </div>
                  <p className="font-mono text-[11px] tracking-wide text-muted-foreground uppercase">
                    {formatSpentAt(expense.spentAt)}
                  </p>
                  {expense.tags.length > 0 ? (
                    <div className="flex flex-wrap gap-1.5">
                      {expense.tags.map((tag) => (
                        <span
                          key={tag}
                          className="inline-flex items-center rounded-sm border border-border px-1.5 py-0.5 text-[10px] font-medium tracking-[0.12em] text-muted-foreground uppercase"
                        >
                          {tag}
                        </span>
                      ))}
                    </div>
                  ) : null}
                </div>

                <div className="flex shrink-0 items-center justify-between gap-3 sm:justify-end">
                  <span
                    className={cn(
                      "font-display text-2xl font-semibold tracking-tight tabular-nums",
                      isCredit && "text-chart-2"
                    )}
                  >
                    {formatDirectionalMoney(
                      expense.amount,
                      expense.currency,
                      expense.type
                    )}
                  </span>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    onClick={() => setPendingDelete(expense)}
                    disabled={isDeleting}
                    aria-label={`Delete ${label}`}
                    className="text-muted-foreground opacity-100 transition-opacity sm:opacity-0 sm:group-hover/row:opacity-100 sm:focus-visible:opacity-100"
                  >
                    {isDeleting ? (
                      <Spinner />
                    ) : (
                      <Trash2Icon aria-hidden="true" />
                    )}
                  </Button>
                </div>
              </li>
            )
          })}
        </ul>
      </div>

      <AlertDialog
        open={pendingDelete !== null}
        onOpenChange={(open) => {
          if (!open) {
            setPendingDelete(null)
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia>
              <ReceiptTextIcon aria-hidden="true" />
            </AlertDialogMedia>
            <AlertDialogTitle>Delete expense?</AlertDialogTitle>
            <AlertDialogDescription>
              This permanently removes{" "}
              <span className="font-medium text-foreground">
                {pendingDelete?.description.trim() || "this expense"}
              </span>
              . This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction variant="destructive" onClick={confirmDelete}>
              Delete expense
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}

function formatSpentAt(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date)
}
