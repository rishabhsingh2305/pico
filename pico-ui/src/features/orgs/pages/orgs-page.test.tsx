import { render, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter, Route, Routes } from "react-router"
import { afterEach, describe, expect, it, vi } from "vitest"

import type { Session } from "@/features/auth/session"
import { RequireSession } from "@/features/auth/auth-provider"
import { OrgsPage } from "@/features/orgs/pages/orgs-page"
import type { Org } from "@/features/orgs/types"
import { AuthTestProvider } from "@/test/auth"

const session: Session = {
  userId: "user-1",
  email: "ada@example.com",
}

const ownedOrg: Org = {
  id: "org-1",
  name: "Acme",
  userId: "user-1",
  tags: ["team"],
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
}

const memberOrg: Org = {
  id: "org-2",
  name: "Beta",
  userId: "user-2",
  tags: ["finance"],
  createdAt: "2026-01-02T00:00:00Z",
  updatedAt: "2026-01-02T00:00:00Z",
}

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  } as Response
}

function renderOrgsPage(initialSession: Session | null = session) {
  return render(
    <AuthTestProvider session={initialSession}>
      <MemoryRouter initialEntries={["/orgs"]}>
        <Routes>
          <Route element={<RequireSession />}>
            <Route path="/orgs" element={<OrgsPage />} />
          </Route>
          <Route path="/login" element={<div>Login destination</div>} />
        </Routes>
      </MemoryRouter>
    </AuthTestProvider>
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe("OrgsPage", () => {
  it("redirects to login without a session", () => {
    renderOrgsPage(null)

    expect(screen.getByText("Login destination")).toBeInTheDocument()
  })

  it("loads and renders organizations", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn<typeof fetch>().mockResolvedValueOnce(
        jsonResponse({
          items: [ownedOrg, memberOrg],
        })
      )
    )

    renderOrgsPage()

    expect(await screen.findByText("Acme")).toBeInTheDocument()
    expect(screen.getByText("Beta")).toBeInTheDocument()
    expect(screen.getByText("Owner")).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "API docs" })).toHaveAttribute(
      "href",
      "/api-docs"
    )
    expect(screen.getByText("Member")).toBeInTheDocument()
    expect(
      screen.queryByRole("button", { name: "Edit tags for Beta" })
    ).not.toBeInTheDocument()
    expect(screen.getByText("ada@example.com")).toBeInTheDocument()
    expect(screen.getByRole("link", { name: /Acme/ })).toHaveAttribute(
      "href",
      "/orgs/org-1"
    )

    const [url] = vi.mocked(fetch).mock.calls[0]
    expect(url).toBe("/api/orgs?userId=user-1&limit=100")
  })

  it("shows an empty state", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn<typeof fetch>().mockResolvedValueOnce(jsonResponse({ items: [] }))
    )

    renderOrgsPage()

    expect(await screen.findByText("No organizations yet")).toBeInTheDocument()
  })

  it("creates an organization with tags and prepends it", async () => {
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
      .mockResolvedValueOnce(jsonResponse(ownedOrg, 201))
    vi.stubGlobal("fetch", fetchMock)
    const userEventInstance = userEvent.setup()

    renderOrgsPage()
    await screen.findByText("No organizations yet")

    await userEventInstance.type(screen.getByLabelText("Name"), "  Acme  ")
    await userEventInstance.type(screen.getByLabelText("Tags"), "team,")
    await userEventInstance.click(
      screen.getByRole("button", { name: "Create organization" })
    )

    expect(await screen.findByText("Acme")).toBeInTheDocument()
    expect(screen.getByText("Owner")).toBeInTheDocument()
    expect(screen.getByText("team")).toBeInTheDocument()
    expect(screen.getByLabelText("Name")).toHaveValue("")
    expect(screen.getByLabelText("Tags")).toHaveValue("")

    const [url, init] = fetchMock.mock.calls[1]
    expect(url).toBe("/api/orgs?userId=user-1")
    expect(init?.method).toBe("POST")
    expect(new Headers(init?.headers).get("Content-Type")).toBe(
      "application/json"
    )
    expect(JSON.parse(String(init?.body))).toEqual({
      name: "Acme",
      tags: ["team"],
    })
  })

  it("filters organizations by tag", async () => {
    const fetchMock = vi.fn<typeof fetch>(async (input, init) => {
      const url = new URL(String(input), "http://localhost")
      const method = init?.method ?? "GET"
      if (method === "GET" && url.pathname === "/api/orgs") {
        const tags = url.searchParams.get("tags")
        return jsonResponse({
          items: tags === "team" ? [ownedOrg] : [ownedOrg, memberOrg],
        })
      }
      return jsonResponse({ error: "not found" }, 404)
    })
    vi.stubGlobal("fetch", fetchMock)
    const userEventInstance = userEvent.setup()

    renderOrgsPage()

    expect(await screen.findByText("Beta")).toBeInTheDocument()
    expect(screen.getByText("Filter tags (any)")).toBeInTheDocument()

    await userEventInstance.click(
      screen.getByRole("button", { name: "Filter by team" })
    )

    await waitFor(() => {
      expect(screen.queryByText("Beta")).not.toBeInTheDocument()
    })
    expect(screen.getByText("Acme")).toBeInTheDocument()

    const urls = fetchMock.mock.calls.map(
      ([input]) => new URL(String(input), "http://localhost")
    )
    expect(urls.some((url) => url.searchParams.get("tags") === "team")).toBe(
      true
    )
  })

  it("shows a filtered empty state", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn<typeof fetch>(async (input, init) => {
        const url = new URL(String(input), "http://localhost")
        const method = init?.method ?? "GET"
        if (method === "GET" && url.pathname === "/api/orgs") {
          const tags = url.searchParams.get("tags")
          return jsonResponse({
            items: tags === "finance" ? [] : [ownedOrg, memberOrg],
          })
        }
        return jsonResponse({ error: "not found" }, 404)
      })
    )
    const userEventInstance = userEvent.setup()

    renderOrgsPage()

    await screen.findByText("Beta")
    await userEventInstance.click(
      screen.getByRole("button", { name: "Filter by finance" })
    )

    expect(
      await screen.findByText("No organizations match these tags")
    ).toBeInTheDocument()
    expect(screen.getByText("Filter tags (any)")).toBeInTheDocument()
    expect(
      screen.getByText("Remove a tag or clear the filter.")
    ).toBeInTheDocument()
  })

  it("edits organization tags from the card", async () => {
    const fetchMock = vi.fn<typeof fetch>(async (input, init) => {
      const url = new URL(String(input), "http://localhost")
      const method = init?.method ?? "GET"
      if (method === "GET" && url.pathname === "/api/orgs") {
        return jsonResponse({ items: [ownedOrg] })
      }
      if (method === "PATCH" && url.pathname === "/api/orgs/org-1") {
        return jsonResponse({ ...ownedOrg, tags: ["ops", "team"] })
      }
      return jsonResponse({ error: "not found" }, 404)
    })
    vi.stubGlobal("fetch", fetchMock)
    const userEventInstance = userEvent.setup()

    renderOrgsPage()
    await screen.findByText("Acme")

    await userEventInstance.click(
      screen.getByRole("button", { name: "Edit tags for Acme" })
    )
    const dialog = await screen.findByRole("dialog", {
      name: "Edit organization tags",
    })
    await userEventInstance.type(within(dialog).getByLabelText("Tags"), "ops,")
    await userEventInstance.click(
      within(dialog).getByRole("button", { name: "Save tags" })
    )

    await waitFor(() => {
      const call = fetchMock.mock.calls.find(
        ([url, init]) =>
          String(url).startsWith("/api/orgs/org-1?") && init?.method === "PATCH"
      )
      expect(call).toBeDefined()
      expect(JSON.parse(String(call?.[1]?.body))).toEqual({
        tags: ["team", "ops"],
      })
    })
  })

  it("validates the organization name before submitting", async () => {
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
    vi.stubGlobal("fetch", fetchMock)
    const userEventInstance = userEvent.setup()

    renderOrgsPage()
    await screen.findByText("No organizations yet")

    await userEventInstance.click(
      screen.getByRole("button", { name: "Create organization" })
    )

    expect(screen.getByText("Enter an organization name.")).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it("shows create errors", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn<typeof fetch>()
        .mockResolvedValueOnce(jsonResponse({ items: [] }))
        .mockResolvedValueOnce(
          jsonResponse({ error: "name must be at most 120 characters" }, 400)
        )
    )
    const userEventInstance = userEvent.setup()

    renderOrgsPage()
    await screen.findByText("No organizations yet")

    await userEventInstance.type(screen.getByLabelText("Name"), "Acme")
    await userEventInstance.click(
      screen.getByRole("button", { name: "Create organization" })
    )

    expect(
      await screen.findByText("name must be at most 120 characters")
    ).toBeInTheDocument()
  })

  it("clears a stale session and redirects to login", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn<typeof fetch>()
        .mockResolvedValueOnce(jsonResponse({ error: "user not found" }, 404))
    )

    renderOrgsPage()

    expect(await screen.findByText("Login destination")).toBeInTheDocument()
  })

  it("shows a list error and retries", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn<typeof fetch>()
        .mockRejectedValueOnce(new TypeError("Failed to fetch"))
        .mockResolvedValueOnce(jsonResponse({ items: [] }))
    )
    const userEventInstance = userEvent.setup()

    renderOrgsPage()

    expect(
      await screen.findByText(
        "Could not reach the server. Check your connection and try again."
      )
    ).toBeInTheDocument()

    await userEventInstance.click(
      screen.getByRole("button", { name: "Try again" })
    )

    expect(await screen.findByText("No organizations yet")).toBeInTheDocument()
    await waitFor(() => {
      expect(vi.mocked(fetch)).toHaveBeenCalledTimes(2)
    })
  })
})
