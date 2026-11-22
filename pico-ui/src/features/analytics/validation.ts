const EVENT_PATTERN = /^[a-z0-9][a-z0-9._-]{0,63}$/
const MAX_EVENTS = 20

export type AnalyticsFormErrors = {
  name?: string
  description?: string
  events?: string
}

export function normalizeAnalyticsEvents(events: string[]): string[] {
  const seen = new Set<string>()
  const normalized: string[] = []
  for (const raw of events) {
    const event = raw.trim().toLowerCase()
    if (event === "" || seen.has(event)) {
      continue
    }
    seen.add(event)
    normalized.push(event)
  }
  return normalized
}

export function validateAnalyticsForm(values: {
  name: string
  description: string
  events: string[]
}): AnalyticsFormErrors {
  const errors: AnalyticsFormErrors = {}
  if (values.name.trim() === "") {
    errors.name = "Enter a name."
  } else if (values.name.trim().length > 120) {
    errors.name = "Name must be at most 120 characters."
  }
  if (values.description.trim().length > 280) {
    errors.description = "Description must be at most 280 characters."
  }
  const events = normalizeAnalyticsEvents(values.events)
  if (events.length === 0) {
    errors.events = "Add at least one event."
  } else if (events.length > MAX_EVENTS) {
    errors.events = "Use at most 20 events."
  } else if (events.some((event) => !EVENT_PATTERN.test(event))) {
    errors.events =
      "Events must use lowercase letters, numbers, dots, underscores, or hyphens."
  }
  return errors
}
