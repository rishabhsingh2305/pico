import { describe, expect, it } from "vitest"

import { canManageShares, canViewShares } from "@/features/sharing/access"

describe("share access", () => {
  it("allows owners to manage and view shares", () => {
    expect(canManageShares(true, [])).toBe(true)
    expect(canViewShares(true, [])).toBe(true)
  })

  it("allows the owner wildcard and share admins to manage", () => {
    expect(canManageShares(false, ["*"])).toBe(true)
    expect(canManageShares(false, ["share.admin"])).toBe(true)
    expect(canManageShares(false, ["share.viewer"])).toBe(false)
  })

  it("allows share viewers to see badges without managing links", () => {
    expect(canViewShares(false, ["share.viewer"])).toBe(true)
    expect(canManageShares(false, ["share.viewer"])).toBe(false)
  })
})
