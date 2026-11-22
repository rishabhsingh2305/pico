import { useId, useState, type FormEvent } from "react"

import { Button } from "@/components/ui/button"
import { DateRangePicker } from "@/components/date-range-picker"
import { Field, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { SelectField } from "@/components/select-field"
import { defaultAnalyticsFilters } from "@/features/analytics/filters"
import type { AnalyticsStatsFilters as Filters } from "@/features/analytics/types"

type AnalyticsFiltersProps = {
  events: string[]
  value: Filters
  onChange: (filters: Filters) => void
}

export function AnalyticsFilters({
  events,
  value,
  onChange,
}: AnalyticsFiltersProps) {
  const id = useId()
  const [draft, setDraft] = useState(value)

  function update<K extends keyof Filters>(key: K, next: Filters[K]) {
    setDraft((current) => ({ ...current, [key]: next }))
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    onChange(draft)
  }

  function handleReset() {
    const next = defaultAnalyticsFilters()
    setDraft(next)
    onChange(next)
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="grid gap-3 border-y border-border py-4 sm:grid-cols-2 lg:grid-cols-4"
      aria-label="Filter analytics"
    >
      <Field>
        <FieldLabel
          htmlFor={`${id}-range`}
          className="text-label text-muted-foreground uppercase"
        >
          Date range
        </FieldLabel>
        <DateRangePicker
          id={`${id}-range`}
          from={draft.from}
          to={draft.to}
          onChange={(range) =>
            setDraft((current) => ({
              ...current,
              from: range.from,
              to: range.to,
            }))
          }
        />
      </Field>

      <Field>
        <FieldLabel
          htmlFor={`${id}-event`}
          className="text-label text-muted-foreground uppercase"
        >
          Event
        </FieldLabel>
        <SelectField
          id={`${id}-event`}
          value={draft.event}
          onValueChange={(next) => update("event", next)}
          options={[
            { value: "", label: "All events" },
            ...events.map((event) => ({ value: event, label: event })),
          ]}
        />
      </Field>

      <Field>
        <FieldLabel
          htmlFor={`${id}-interval`}
          className="text-label text-muted-foreground uppercase"
        >
          Group by
        </FieldLabel>
        <SelectField
          id={`${id}-interval`}
          value={draft.interval}
          onValueChange={(next) =>
            update("interval", next === "hour" ? "hour" : "day")
          }
          options={[
            { value: "day", label: "Day" },
            { value: "hour", label: "Hour" },
          ]}
        />
      </Field>

      <Field>
        <FieldLabel
          htmlFor={`${id}-device`}
          className="text-label text-muted-foreground uppercase"
        >
          Device
        </FieldLabel>
        <SelectField
          id={`${id}-device`}
          value={draft.device}
          onValueChange={(next) => update("device", next)}
          options={[
            { value: "", label: "All devices" },
            { value: "desktop", label: "Desktop" },
            { value: "mobile", label: "Mobile" },
            { value: "tablet", label: "Tablet" },
            { value: "bot", label: "Bot" },
            { value: "unknown", label: "Unknown" },
          ]}
        />
      </Field>

      <Field>
        <FieldLabel
          htmlFor={`${id}-country`}
          className="text-label text-muted-foreground uppercase"
        >
          Country
        </FieldLabel>
        <Input
          id={`${id}-country`}
          value={draft.country}
          onChange={(event) =>
            update("country", event.target.value.toUpperCase())
          }
          placeholder="IN"
          maxLength={2}
          autoComplete="off"
        />
      </Field>

      <Field>
        <FieldLabel
          htmlFor={`${id}-referrer`}
          className="text-label text-muted-foreground uppercase"
        >
          Referrer
        </FieldLabel>
        <Input
          id={`${id}-referrer`}
          value={draft.referrer}
          onChange={(event) => update("referrer", event.target.value)}
          placeholder="twitter.com"
          autoComplete="off"
        />
      </Field>

      <Field>
        <FieldLabel
          htmlFor={`${id}-bot`}
          className="text-label text-muted-foreground uppercase"
        >
          Traffic
        </FieldLabel>
        <SelectField
          id={`${id}-bot`}
          value={draft.bot}
          onValueChange={(next) => update("bot", next)}
          options={[
            { value: "", label: "All traffic" },
            { value: "false", label: "Humans" },
            { value: "true", label: "Bots" },
          ]}
        />
      </Field>

      <div className="flex items-end gap-2">
        <Button type="submit">Apply</Button>
        <Button type="button" variant="ghost" onClick={handleReset}>
          Reset
        </Button>
      </div>
    </form>
  )
}
