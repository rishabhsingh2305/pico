import { useState } from "react"
import { CalendarIcon } from "lucide-react"
import type { DateRange } from "react-day-picker"

import { Button } from "@/components/ui/button"
import { Calendar } from "@/components/ui/calendar"
import {
  Popover,
  PopoverContent,
  PopoverTitle,
  PopoverTrigger,
} from "@/components/ui/popover"
import { useIsMobile } from "@/hooks/use-mobile"
import {
  DATE_RANGE_PRESETS,
  dateRangeFromValues,
  findDateRangePreset,
  formatDateRangeLabel,
  toDateRangeValue,
  type DateRangeValue,
} from "@/lib/date-range"
import { cn } from "cn"

type DateRangePickerProps = {
  id?: string
  from: string
  to: string
  onChange: (value: DateRangeValue) => void
  today?: Date
  className?: string
}

export function DateRangePicker({
  id,
  from,
  to,
  onChange,
  today,
  className,
}: DateRangePickerProps) {
  const isMobile = useIsMobile()
  const [open, setOpen] = useState(false)
  const [draft, setDraft] = useState<DateRange>(() =>
    dateRangeFromValues(from, to)
  )

  const reference = today ?? new Date()
  const activePreset = findDateRangePreset(from, to, reference)
  const label = activePreset?.label ?? formatDateRangeLabel(from, to)

  function handleOpenChange(nextOpen: boolean) {
    if (nextOpen) {
      setDraft(dateRangeFromValues(from, to))
    }
    setOpen(nextOpen)
  }

  function handleSelect(range: DateRange | undefined) {
    setDraft(range ?? { from: undefined })
    if (range?.from && range.to) {
      onChange(toDateRangeValue(range))
      setOpen(false)
    }
  }

  function handlePreset(value: DateRangeValue) {
    onChange(value)
    setDraft(dateRangeFromValues(value.from, value.to))
    setOpen(false)
  }

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger
        render={
          <Button
            id={id}
            type="button"
            variant="outline"
            aria-label={`Date range: ${label}`}
            className={cn(
              "w-full min-w-0 justify-start gap-2 font-normal",
              from === "" && to === "" && "text-muted-foreground",
              className
            )}
          />
        }
      >
        <CalendarIcon data-icon="inline-start" aria-hidden="true" />
        <span className="truncate">{label}</span>
      </PopoverTrigger>
      <PopoverContent
        align="start"
        className="w-auto max-w-[calc(100vw-2rem)] p-0"
      >
        <PopoverTitle className="sr-only">Choose a date range</PopoverTitle>
        <div className="flex flex-col sm:flex-row">
          <div className="flex shrink-0 flex-wrap gap-1 border-b p-2 sm:w-40 sm:flex-col sm:flex-nowrap sm:border-r sm:border-b-0">
            {DATE_RANGE_PRESETS.map((preset) => {
              const active = activePreset?.id === preset.id
              return (
                <Button
                  key={preset.id}
                  type="button"
                  variant={active ? "secondary" : "ghost"}
                  size="sm"
                  className="justify-start whitespace-nowrap"
                  aria-pressed={active}
                  onClick={() => handlePreset(preset.resolve(reference))}
                >
                  {preset.label}
                </Button>
              )
            })}
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="justify-start whitespace-nowrap"
              disabled={from === "" && to === ""}
              onClick={() => handlePreset({ from: "", to: "" })}
            >
              Clear dates
            </Button>
          </div>
          <div className="p-2">
            <Calendar
              mode="range"
              selected={draft}
              onSelect={handleSelect}
              defaultMonth={draft.from ?? reference}
              numberOfMonths={isMobile ? 1 : 2}
              resetOnSelect
            />
          </div>
        </div>
      </PopoverContent>
    </Popover>
  )
}
