import { useId, useState, type FormEvent } from "react"
import { CircleAlertIcon } from "lucide-react"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldError,
  FieldLabel,
  FieldLegend,
  FieldSet,
  FieldTitle,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"
import { Spinner } from "@/components/ui/spinner"
import { Textarea } from "@/components/ui/textarea"
import { isSessionInvalidError } from "@/features/auth/session"
import { TagInput } from "@/components/tag-input"
import { createExpense } from "@/features/expenses/api"
import { expenseErrorMessage } from "@/features/expenses/error-message"
import type { Expense, ExpenseType } from "@/features/expenses/types"
import {
  nowLocalDateTime,
  validateExpense,
  type ExpenseErrors,
} from "@/features/expenses/validation"

type CreateExpenseFormProps = {
  userId: string
  orgId: string
  projectId: string
  currency: string
  onCreated: (expense: Expense) => void
  onCancel?: () => void
  onSessionInvalid: () => void
}

export function CreateExpenseForm({
  userId,
  orgId,
  projectId,
  currency,
  onCreated,
  onCancel,
  onSessionInvalid,
}: CreateExpenseFormProps) {
  const id = useId()
  const typeId = `${id}-type`
  const debitId = `${id}-debit`
  const creditId = `${id}-credit`
  const amountId = `${id}-amount`
  const spentAtId = `${id}-spent-at`
  const descriptionId = `${id}-description`
  const tagsId = `${id}-tags`

  const [type, setType] = useState<ExpenseType>("debit")
  const [amount, setAmount] = useState("")
  const [description, setDescription] = useState("")
  const [tags, setTags] = useState<string[]>([])
  const [spentAt, setSpentAt] = useState(nowLocalDateTime)
  const [errors, setErrors] = useState<ExpenseErrors>({})
  const [formError, setFormError] = useState("")
  const [isSubmitting, setIsSubmitting] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (isSubmitting) {
      return
    }

    const result = validateExpense(
      {
        projectId,
        type,
        amount,
        description,
        tags,
        spentAt,
      },
      currency
    )
    setErrors(result.errors)
    setFormError("")
    if (Object.keys(result.errors).length > 0) {
      return
    }
    if (
      result.type === undefined ||
      result.amountMinorUnits === undefined ||
      result.spentAtIso === undefined
    ) {
      return
    }

    setIsSubmitting(true)
    try {
      const expense = await createExpense(userId, orgId, {
        projectId,
        type: result.type,
        amount: result.amountMinorUnits,
        description: description.trim(),
        tags: result.tags ?? [],
        spentAt: result.spentAtIso,
      })
      setType("debit")
      setAmount("")
      setDescription("")
      setTags([])
      setSpentAt(nowLocalDateTime())
      setIsSubmitting(false)
      onCreated(expense)
    } catch (error) {
      if (isSessionInvalidError(error)) {
        onSessionInvalid()
        return
      }
      setFormError(expenseErrorMessage(error))
      setIsSubmitting(false)
    }
  }

  function clearError(field: keyof ExpenseErrors) {
    if (errors[field]) {
      setErrors((current) => ({ ...current, [field]: undefined }))
    }
    if (formError) {
      setFormError("")
    }
  }

  return (
    <form
      onSubmit={handleSubmit}
      noValidate
      aria-busy={isSubmitting}
      className="flex flex-col gap-4"
    >
      <FieldSet>
        <FieldLegend variant="label">Type</FieldLegend>
        <RadioGroup
          id={typeId}
          value={type}
          onValueChange={(value) => {
            if (value === "debit" || value === "credit") {
              setType(value)
              clearError("type")
            }
          }}
          disabled={isSubmitting}
          aria-invalid={Boolean(errors.type)}
          aria-describedby={errors.type ? `${id}-type-error` : undefined}
          className="grid grid-cols-2 gap-2"
        >
          <FieldLabel
            htmlFor={debitId}
            className="w-full cursor-pointer items-start rounded-lg border p-3"
          >
            <RadioGroupItem id={debitId} value="debit" />
            <FieldContent>
              <FieldTitle>Debit</FieldTitle>
              <FieldDescription>Money out</FieldDescription>
            </FieldContent>
          </FieldLabel>
          <FieldLabel
            htmlFor={creditId}
            className="w-full cursor-pointer items-start rounded-lg border p-3"
          >
            <RadioGroupItem id={creditId} value="credit" />
            <FieldContent>
              <FieldTitle>Credit</FieldTitle>
              <FieldDescription>Money in</FieldDescription>
            </FieldContent>
          </FieldLabel>
        </RadioGroup>
        <FieldError id={`${id}-type-error`}>{errors.type}</FieldError>
      </FieldSet>

      <Field data-invalid={Boolean(errors.amount)}>
        <FieldLabel htmlFor={amountId}>Amount</FieldLabel>
        <Input
          id={amountId}
          name="amount"
          type="text"
          inputMode="decimal"
          placeholder="12.99"
          value={amount}
          onChange={(event) => {
            setAmount(event.target.value)
            clearError("amount")
          }}
          disabled={isSubmitting}
          aria-invalid={Boolean(errors.amount)}
          aria-describedby={errors.amount ? `${id}-amount-error` : undefined}
        />
        <FieldError id={`${id}-amount-error`}>{errors.amount}</FieldError>
      </Field>

      <Field data-invalid={Boolean(errors.spentAt)}>
        <FieldLabel htmlFor={spentAtId}>Date</FieldLabel>
        <Input
          id={spentAtId}
          name="spentAt"
          type="datetime-local"
          value={spentAt}
          onChange={(event) => {
            setSpentAt(event.target.value)
            clearError("spentAt")
          }}
          disabled={isSubmitting}
          aria-invalid={Boolean(errors.spentAt)}
          aria-describedby={errors.spentAt ? `${id}-spent-at-error` : undefined}
        />
        <FieldError id={`${id}-spent-at-error`}>{errors.spentAt}</FieldError>
      </Field>

      <Field data-invalid={Boolean(errors.description)}>
        <FieldLabel htmlFor={descriptionId}>Description</FieldLabel>
        <Textarea
          id={descriptionId}
          name="description"
          placeholder={
            type === "credit" ? "Client payment" : "Lunch with the team"
          }
          rows={2}
          value={description}
          onChange={(event) => {
            setDescription(event.target.value)
            clearError("description")
          }}
          disabled={isSubmitting}
          aria-invalid={Boolean(errors.description)}
          aria-describedby={
            errors.description ? `${id}-description-error` : undefined
          }
        />
        <FieldError id={`${id}-description-error`}>
          {errors.description}
        </FieldError>
      </Field>

      <Field data-invalid={Boolean(errors.tags)}>
        <FieldLabel htmlFor={tagsId}>Tags</FieldLabel>
        <TagInput
          id={tagsId}
          name="tags"
          value={tags}
          onChange={(nextTags) => {
            setTags(nextTags)
            clearError("tags")
          }}
          placeholder="food, travel"
          disabled={isSubmitting}
          invalid={Boolean(errors.tags)}
          describedBy={errors.tags ? `${id}-tags-error` : `${id}-tags-help`}
        />
        <FieldDescription id={`${id}-tags-help`}>
          Type a tag and press comma or Enter. Up to 20 tags.
        </FieldDescription>
        <FieldError id={`${id}-tags-error`}>{errors.tags}</FieldError>
      </Field>

      {formError ? (
        <Alert variant="destructive">
          <CircleAlertIcon aria-hidden="true" />
          <AlertDescription>{formError}</AlertDescription>
        </Alert>
      ) : null}

      <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
        {onCancel ? (
          <Button
            type="button"
            variant="outline"
            onClick={onCancel}
            disabled={isSubmitting}
          >
            Cancel
          </Button>
        ) : null}
        <Button
          type="submit"
          disabled={isSubmitting}
          className="w-full sm:w-auto"
        >
          {isSubmitting ? (
            <>
              <Spinner data-icon="inline-start" />
              Adding...
            </>
          ) : (
            "Add expense"
          )}
        </Button>
      </div>
    </form>
  )
}
