import {
  lastThreeMonthsRange,
  monthRange,
  previousMonthRange,
  yearRange,
} from "@/features/expenses/months"

export type DatePreset =
  "all" | "this-month" | "last-month" | "last-3-months" | "this-year" | "custom"

export type FilterDraft = {
  preset: DatePreset
  customFrom: string
  customTo: string
  tags: string[]
}

export type AppliedFilters = FilterDraft & {
  from?: string
  to?: string
  dateLabel: string
}

export const defaultFilters: AppliedFilters = {
  preset: "all",
  customFrom: "",
  customTo: "",
  tags: [],
  dateLabel: "All time",
}

export type ResolveFiltersResult =
  { ok: true; filters: AppliedFilters } | { ok: false; error: string }

export function resolveFilters(draft: FilterDraft): ResolveFiltersResult {
  const tags = [...draft.tags].sort((a, b) => a.localeCompare(b))

  if (draft.preset === "all") {
    return {
      ok: true,
      filters: {
        ...draft,
        tags,
        from: undefined,
        to: undefined,
        dateLabel: "All time",
      },
    }
  }

  if (draft.preset === "this-month") {
    const range = monthRange()
    return {
      ok: true,
      filters: {
        ...draft,
        tags,
        from: range.from,
        to: range.to,
        dateLabel: "This month",
      },
    }
  }

  if (draft.preset === "last-month") {
    const range = previousMonthRange()
    return {
      ok: true,
      filters: {
        ...draft,
        tags,
        from: range.from,
        to: range.to,
        dateLabel: "Last month",
      },
    }
  }

  if (draft.preset === "last-3-months") {
    const range = lastThreeMonthsRange()
    return {
      ok: true,
      filters: {
        ...draft,
        tags,
        from: range.from,
        to: range.to,
        dateLabel: "Last 3 months",
      },
    }
  }

  if (draft.preset === "this-year") {
    const range = yearRange()
    return {
      ok: true,
      filters: {
        ...draft,
        tags,
        from: range.from,
        to: range.to,
        dateLabel: "This year",
      },
    }
  }

  if (!draft.customFrom && !draft.customTo) {
    return { ok: false, error: "Choose a start or end date." }
  }

  const from = draft.customFrom ? dateToStartOfDay(draft.customFrom) : null
  const to = draft.customTo ? dateToEndOfDay(draft.customTo) : null

  if (draft.customFrom && from === null) {
    return { ok: false, error: "Choose a valid start date." }
  }
  if (draft.customTo && to === null) {
    return { ok: false, error: "Choose a valid end date." }
  }
  if (from && to && new Date(from) > new Date(to)) {
    return { ok: false, error: "The start date must be before the end date." }
  }

  const dateLabel =
    draft.customFrom && draft.customTo
      ? `${formatFilterDate(draft.customFrom)} – ${formatFilterDate(draft.customTo)}`
      : draft.customFrom
        ? `From ${formatFilterDate(draft.customFrom)}`
        : `Until ${formatFilterDate(draft.customTo)}`

  return {
    ok: true,
    filters: {
      ...draft,
      tags,
      from: from ?? undefined,
      to: to ?? undefined,
      dateLabel,
    },
  }
}

export function activeFilterCount(filters: AppliedFilters): number {
  return filters.tags.length + (filters.preset !== "all" ? 1 : 0)
}

function dateToStartOfDay(value: string): string | null {
  const date = new Date(`${value}T00:00:00`)
  return Number.isNaN(date.getTime()) ? null : date.toISOString()
}

function dateToEndOfDay(value: string): string | null {
  const date = new Date(`${value}T23:59:59.999`)
  return Number.isNaN(date.getTime()) ? null : date.toISOString()
}

function formatFilterDate(value: string): string {
  const date = new Date(`${value}T12:00:00`)
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  }).format(date)
}
