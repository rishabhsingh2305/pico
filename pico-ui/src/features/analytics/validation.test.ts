import { describe, expect, it } from "vitest"

import {
  normalizeAnalyticsEvents,
  validateAnalyticsForm,
} from "@/features/analytics/validation"

describe("analytics validation", () => {
  it("normalizes events to lowercase and removes duplicates", () => {
    expect(
      normalizeAnalyticsEvents([" View", "click", "VIEW", "", "sign-up"])
    ).toEqual(["view", "click", "sign-up"])
  })

  it("requires a name and at least one event", () => {
    expect(
      validateAnalyticsForm({ name: " ", description: "", events: [] })
    ).toEqual({
      name: "Enter a name.",
      events: "Add at least one event.",
    })
  })

  it("rejects invalid event names", () => {
    expect(
      validateAnalyticsForm({
        name: "Homepage",
        description: "",
        events: ["bad event"],
      })
    ).toEqual({
      events:
        "Events must use lowercase letters, numbers, dots, underscores, or hyphens.",
    })
  })

  it("caps names, descriptions, and event counts", () => {
    expect(
      validateAnalyticsForm({
        name: "a".repeat(121),
        description: "a".repeat(281),
        events: Array.from({ length: 21 }, (_, index) => `event-${index}`),
      })
    ).toEqual({
      name: "Name must be at most 120 characters.",
      description: "Description must be at most 280 characters.",
      events: "Use at most 20 events.",
    })
  })

  it("accepts a valid form", () => {
    expect(
      validateAnalyticsForm({
        name: "Homepage CTA",
        description: "Top button",
        events: ["view", "click"],
      })
    ).toEqual({})
  })
})
