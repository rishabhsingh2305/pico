import { afterEach, describe, expect, it } from "vitest"

import {
  acquireShareLink,
  clearShareToken,
  getShareToken,
  parseShareLink,
  setShareToken,
  stripShareFromURL,
} from "@/lib/share-token"

afterEach(() => {
  clearShareToken()
  window.history.replaceState(null, "", "/")
})

describe("share token", () => {
  it("parses a share link from the fragment and query", () => {
    window.history.replaceState(
      null,
      "",
      "/shared?orgId=org-1&resourceType=expense&resourceId=expense-1&projectId=project-1#share=raw-token"
    )

    expect(parseShareLink()).toEqual({
      token: "raw-token",
      orgId: "org-1",
      resourceType: "expense",
      resourceId: "expense-1",
      projectId: "project-1",
    })
  })

  it("acquires the link, stores the token, and strips the fragment", () => {
    window.history.replaceState(
      null,
      "",
      "/shared?orgId=org-1&resourceType=project&resourceId=project-1#share=raw-token"
    )

    const link = acquireShareLink()

    expect(link?.token).toBe("raw-token")
    expect(getShareToken()).toBe("raw-token")
    expect(window.location.hash).toBe("")
  })

  it("falls back to session storage after the fragment is removed", () => {
    window.history.replaceState(
      null,
      "",
      "/shared?orgId=org-1&resourceType=project&resourceId=project-1"
    )
    setShareToken("stored-token")

    expect(parseShareLink()).toEqual({
      token: "stored-token",
      orgId: "org-1",
      resourceType: "project",
      resourceId: "project-1",
      projectId: undefined,
    })
  })

  it("strips only the share value from the fragment", () => {
    window.history.replaceState(null, "", "/shared?orgId=org-1#share=raw-token")

    stripShareFromURL()

    expect(window.location.hash).toBe("")
    expect(window.location.search).toBe("?orgId=org-1")
  })

  it("returns null when there is no token", () => {
    window.history.replaceState(
      null,
      "",
      "/shared?orgId=org-1&resourceType=project&resourceId=project-1"
    )
    expect(parseShareLink()).toBeNull()
    expect(getShareToken()).toBeNull()
  })
})
