import { useId, useState } from "react"
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
import {
  resolveFilters,
  type AppliedFilters,
  type DatePreset,
} from "@/features/expenses/filters"

type FilterSheetProps = {
  availableTags: string[]
  initialFilters: AppliedFilters
  onApply: (filters: AppliedFilters) => void
  onOpenChange: (open: boolean) => void
}

const presets: Array<{ value: DatePreset; label: string }> = [
  { value: "all", label: "All time" },
  { value: "this-month", label: "This month" },
  { value: "last-month", label: "Last month" },
  { value: "last-3-months", label: "Last 3 months" },
  { value: "this-year", label: "This year" },
  { value: "custom", label: "Custom" },
]

export function FilterSheet({
  availableTags,
  initialFilters,
  onApply,
  onOpenChange,
}: FilterSheetProps) {
  const id = useId()
  const fromId = `${id}-from`
  const toId = `${id}-to`

  const [preset, setPreset] = useState<DatePreset>(initialFilters.preset)
  const [customFrom, setCustomFrom] = useState(initialFilters.customFrom)
  const [customTo, setCustomTo] = useState(initialFilters.customTo)
  const [tags, setTags] = useState<string[]>(initialFilters.tags)
  const [error, setError] = useState("")

  function toggleTag(tag: string) {
    setTags((current) => {
      if (current.includes(tag)) {
        return current.filter((item) => item !== tag)
      }
      return [...current, tag].sort((a, b) => a.localeCompare(b))
    })
    setError("")
  }

  function reset() {
    setPreset("all")
    setCustomFrom("")
    setCustomTo("")
    setTags([])
    setError("")
  }

  function apply() {
    const result = resolveFilters({ preset, customFrom, customTo, tags })
    if (!result.ok) {
      setError(result.error)
      return
    }
    onApply(result.filters)
  }

  return (
    <Dialog open onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Filters</DialogTitle>
          <DialogDescription>
            Narrow the expense list by date or tags.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-5">
          <div className="flex flex-col gap-2">
            <span className="text-sm font-semibold">Date</span>
            <div className="flex flex-wrap gap-2">
              {presets.map((item) => (
                <Button
                  key={item.value}
                  type="button"
                  size="sm"
                  variant={preset === item.value ? "default" : "outline"}
                  aria-pressed={preset === item.value}
                  onClick={() => {
                    setPreset(item.value)
                    setError("")
                  }}
                >
                  {item.label}
                </Button>
              ))}
            </div>

            {preset === "custom" ? (
              <div className="grid grid-cols-2 gap-2">
                <div className="flex flex-col gap-1">
                  <Label htmlFor={fromId}>From</Label>
                  <Input
                    id={fromId}
                    type="date"
                    value={customFrom}
                    onChange={(event) => {
                      setCustomFrom(event.target.value)
                      setError("")
                    }}
                  />
                </div>
                <div className="flex flex-col gap-1">
                  <Label htmlFor={toId}>To</Label>
                  <Input
                    id={toId}
                    type="date"
                    value={customTo}
                    onChange={(event) => {
                      setCustomTo(event.target.value)
                      setError("")
                    }}
                  />
                </div>
              </div>
            ) : null}
          </div>

          <div className="flex flex-col gap-2">
            <span className="text-sm font-semibold">Tags</span>
            <p className="text-xs text-muted-foreground">
              Expenses must have all selected tags.
            </p>
            {availableTags.length === 0 ? (
              <p className="text-sm text-muted-foreground">No tags yet.</p>
            ) : (
              <div className="flex flex-wrap gap-2">
                {availableTags.map((tag) => (
                  <Button
                    key={tag}
                    type="button"
                    size="sm"
                    variant={tags.includes(tag) ? "default" : "outline"}
                    aria-pressed={tags.includes(tag)}
                    onClick={() => toggleTag(tag)}
                  >
                    {tag}
                  </Button>
                ))}
              </div>
            )}
            <p className="text-[0.65rem] text-muted-foreground">
              Tags come from the latest 100 expenses.
            </p>
          </div>

          {error ? (
            <Alert variant="destructive">
              <CircleAlertIcon aria-hidden="true" />
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          ) : null}
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={reset}>
            Reset
          </Button>
          <Button type="button" onClick={apply}>
            Apply filters
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
