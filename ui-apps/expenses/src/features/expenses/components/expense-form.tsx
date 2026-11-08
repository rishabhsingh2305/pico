import { useId, useState, type FormEvent } from "react"
import { CircleAlertIcon } from "lucide-react"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Spinner } from "@/components/ui/spinner"
import { createExpense } from "@/features/expenses/api"
import { TagInput } from "@/features/expenses/components/tag-input"
import { expenseErrorMessage } from "@/features/expenses/error-message"
import {
  dateToSpentAt,
  toMinorUnits,
  todayInputValue,
} from "@/features/expenses/format"
import { MAX_TAGS, normalizeTags } from "@/features/expenses/tags"
import type { ExpenseType } from "@/features/expenses/types"
import { useExpenses } from "@/features/expenses/expenses-provider"
import { byteLength } from "@/lib/text"

const DESCRIPTION_MAX_BYTES = 1000

type ExpenseFormProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: string
  currency: string
  onCreated: () => void
}

type Errors = {
  amount?: string
  description?: string
  date?: string
}

export function ExpenseForm({
  open,
  onOpenChange,
  projectId,
  currency,
  onCreated,
}: ExpenseFormProps) {
  const id = useId()
  const amountId = `${id}-amount`
  const descriptionId = `${id}-description`
  const tagsId = `${id}-tags`
  const dateId = `${id}-date`
  const { orgId } = useExpenses()

  const [type, setType] = useState<ExpenseType>("debit")
  const [amount, setAmount] = useState("")
  const [description, setDescription] = useState("")
  const [date, setDate] = useState(todayInputValue)
  const [tags, setTags] = useState<string[]>([])
  const [tagInput, setTagInput] = useState("")
  const [tagsError, setTagsError] = useState("")
  const [errors, setErrors] = useState<Errors>({})
  const [formError, setFormError] = useState("")
  const [isSubmitting, setIsSubmitting] = useState(false)

  function reset() {
    setType("debit")
    setAmount("")
    setDescription("")
    setDate(todayInputValue())
    setTags([])
    setTagInput("")
    setTagsError("")
    setErrors({})
    setFormError("")
    setIsSubmitting(false)
  }

  function handleOpenChange(next: boolean) {
    if (!next) {
      reset()
    }
    onOpenChange(next)
  }

  function addTags(raw: string) {
    const result = normalizeTags([...tags, ...raw.split(/[\s,]+/)])
    if (!result.ok) {
      setTagsError(result.error)
      return
    }
    setTags(result.tags)
    setTagsError("")
    setTagInput("")
  }

  function removeTag(tag: string) {
    setTags((current) => current.filter((item) => item !== tag))
    setTagsError("")
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (isSubmitting) {
      return
    }

    const nextErrors: Errors = {}
    const minor = toMinorUnits(amount, currency)
    if (minor === null) {
      nextErrors.amount = "Enter an amount greater than zero."
    }
    const spentAt = dateToSpentAt(date)
    if (spentAt === null) {
      nextErrors.date = "Choose a date."
    }
    if (byteLength(description.trim()) > DESCRIPTION_MAX_BYTES) {
      nextErrors.description = "Description must be at most 1000 characters."
    }

    const finalTags = normalizeTags([...tags, ...tagInput.split(/[\s,]+/)])
    if (finalTags.ok) {
      setTagsError("")
    } else {
      setTagsError(finalTags.error)
    }

    setErrors(nextErrors)
    setFormError("")
    if (
      Object.keys(nextErrors).length > 0 ||
      minor === null ||
      spentAt === null ||
      !finalTags.ok
    ) {
      return
    }

    setIsSubmitting(true)
    try {
      await createExpense(orgId, {
        projectId,
        type,
        amount: minor,
        description: description.trim(),
        tags: finalTags.tags.sort(),
        spentAt,
      })
      setIsSubmitting(false)
      onCreated()
      handleOpenChange(false)
    } catch (caught) {
      setFormError(expenseErrorMessage(caught))
      setIsSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add expense</DialogTitle>
          <DialogDescription>
            Record money you spent or received.
          </DialogDescription>
        </DialogHeader>

        <form
          onSubmit={handleSubmit}
          noValidate
          aria-busy={isSubmitting}
          className="flex flex-col gap-4"
        >
          <div className="grid grid-cols-2 gap-2">
            <Button
              type="button"
              variant={type === "debit" ? "default" : "outline"}
              aria-pressed={type === "debit"}
              onClick={() => setType("debit")}
              disabled={isSubmitting}
            >
              Spent
            </Button>
            <Button
              type="button"
              variant={type === "credit" ? "default" : "outline"}
              aria-pressed={type === "credit"}
              onClick={() => setType("credit")}
              disabled={isSubmitting}
            >
              Received
            </Button>
          </div>

          <div className="flex flex-col gap-2">
            <Label htmlFor={amountId}>Amount ({currency})</Label>
            <Input
              id={amountId}
              name="amount"
              type="text"
              inputMode="decimal"
              autoFocus
              placeholder="0.00"
              className="h-11 font-display text-lg"
              value={amount}
              onChange={(event) => {
                setAmount(event.target.value)
                if (errors.amount) {
                  setErrors((current) => ({ ...current, amount: undefined }))
                }
                if (formError) {
                  setFormError("")
                }
              }}
              disabled={isSubmitting}
              aria-invalid={Boolean(errors.amount)}
              aria-describedby={
                errors.amount ? `${id}-amount-error` : undefined
              }
            />
            {errors.amount ? (
              <p id={`${id}-amount-error`} className="text-sm text-destructive">
                {errors.amount}
              </p>
            ) : null}
          </div>

          <div className="flex flex-col gap-2">
            <Label htmlFor={descriptionId}>What was it?</Label>
            <Input
              id={descriptionId}
              name="description"
              type="text"
              placeholder="Lunch, rent, salary..."
              className="h-11"
              value={description}
              onChange={(event) => {
                setDescription(event.target.value)
                if (errors.description) {
                  setErrors((current) => ({
                    ...current,
                    description: undefined,
                  }))
                }
                if (formError) {
                  setFormError("")
                }
              }}
              disabled={isSubmitting}
              aria-invalid={Boolean(errors.description)}
              aria-describedby={
                errors.description ? `${id}-description-error` : undefined
              }
            />
            {errors.description ? (
              <p
                id={`${id}-description-error`}
                className="text-sm text-destructive"
              >
                {errors.description}
              </p>
            ) : null}
          </div>

          <div className="flex flex-col gap-2">
            <Label htmlFor={tagsId}>Tags</Label>
            <TagInput
              id={tagsId}
              tags={tags}
              input={tagInput}
              onInputChange={(value) => {
                setTagInput(value)
                if (tagsError) {
                  setTagsError("")
                }
                if (formError) {
                  setFormError("")
                }
              }}
              onAdd={addTags}
              onRemove={removeTag}
              disabled={isSubmitting}
              error={tagsError}
              describedBy={
                tagsError
                  ? `${id}-tags-hint ${id}-tags-error`
                  : `${id}-tags-hint`
              }
            />
            <p id={`${id}-tags-hint`} className="text-xs text-muted-foreground">
              Press Enter, space, or comma to add. Up to {MAX_TAGS}.
            </p>
            {tagsError ? (
              <p id={`${id}-tags-error`} className="text-sm text-destructive">
                {tagsError}
              </p>
            ) : null}
          </div>

          <div className="flex flex-col gap-2">
            <Label htmlFor={dateId}>Date</Label>
            <Input
              id={dateId}
              name="date"
              type="date"
              className="h-11"
              value={date}
              onChange={(event) => {
                setDate(event.target.value)
                if (errors.date) {
                  setErrors((current) => ({ ...current, date: undefined }))
                }
                if (formError) {
                  setFormError("")
                }
              }}
              disabled={isSubmitting}
              aria-invalid={Boolean(errors.date)}
              aria-describedby={errors.date ? `${id}-date-error` : undefined}
            />
            {errors.date ? (
              <p id={`${id}-date-error`} className="text-sm text-destructive">
                {errors.date}
              </p>
            ) : null}
          </div>

          {formError ? (
            <Alert variant="destructive">
              <CircleAlertIcon aria-hidden="true" />
              <AlertDescription>{formError}</AlertDescription>
            </Alert>
          ) : null}

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => handleOpenChange(false)}
              disabled={isSubmitting}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? (
                <>
                  <Spinner data-icon="inline-start" />
                  Adding...
                </>
              ) : (
                "Add expense"
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
