import { describe, expect, it } from "vitest"

import { hasFeatureRole } from "@/lib/roles"

describe("hasFeatureRole", () => {
  it("accepts the owner wildcard", () => {
    expect(hasFeatureRole(["*"], "link")).toBe(true)
  })

  it("matches any role in the feature namespace", () => {
    expect(hasFeatureRole(["link.viewer"], "link")).toBe(true)
    expect(hasFeatureRole(["link.editor"], "link")).toBe(true)
    expect(hasFeatureRole(["project.editor"], "link")).toBe(false)
    expect(hasFeatureRole([], "link")).toBe(false)
  })
})
