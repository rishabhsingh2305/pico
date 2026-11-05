import { useId, useState, type FormEvent } from "react"
import { CircleAlertIcon } from "lucide-react"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Field, FieldLabel } from "@/components/ui/field"
import { DateRangePicker } from "@/components/date-range-picker"
import { TagInput } from "@/components/tag-input"
import {
  parseExpenseFilters,
  type ExpenseFilters,
} from "@/features/expenses/filters"

type ExpenseFilterFormProps = {
  onApply: (filters: ExpenseFilters) => void
  onClear: () => void
}

export function ExpenseFilterForm({
  onApply,
  onClear,
}: ExpenseFilterFormProps) {
  const id = useId()
  const tagsId = `${id}-tags`
  const rangeId = `${id}-range`

  const [tags, setTags] = useState<string[]>([])
  const [from, setFrom] = useState("")
  const [to, setTo] = useState("")
  const [error, setError] = useState("")
  const [tagInputKey, setTagInputKey] = useState(0)

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    const result = parseExpenseFilters({ tags, from, to })
    if (!result.filters) {
      setError(result.error ?? "Choose valid filters.")
      return
    }

    setError("")
    onApply(result.filters)
  }

  function handleClear() {
    setTags([])
    setFrom("")
    setTo("")
    setError("")
    setTagInputKey((current) => current + 1)
    onClear()
  }

  return (
    <form
      onSubmit={handleSubmit}
      noValidate
      aria-label="Filter expenses"
      className="flex flex-col gap-3 border-y border-border py-4"
    >
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
        <Field className="sm:min-w-0 sm:flex-1">
          <FieldLabel
            htmlFor={tagsId}
            className="text-label text-muted-foreground uppercase"
          >
            Filter tags (all)
          </FieldLabel>
          <TagInput
            key={tagInputKey}
            id={tagsId}
            name="tags"
            value={tags}
            onChange={(nextTags) => {
              setTags(nextTags)
              setError("")
            }}
            placeholder="food, travel"
          />
        </Field>

        <Field className="sm:w-56">
          <FieldLabel
            htmlFor={rangeId}
            className="text-label text-muted-foreground uppercase"
          >
            Date range
          </FieldLabel>
          <DateRangePicker
            id={rangeId}
            from={from}
            to={to}
            onChange={(range) => {
              setFrom(range.from)
              setTo(range.to)
              setError("")
            }}
          />
        </Field>

        <div className="flex gap-2">
          <Button type="submit">Apply</Button>
          <Button type="button" variant="ghost" onClick={handleClear}>
            Clear
          </Button>
        </div>
      </div>

      {error ? (
        <Alert variant="destructive">
          <CircleAlertIcon aria-hidden="true" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}
    </form>
  )
}
