import { render, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter, Route, Routes } from "react-router"
import { afterEach, describe, expect, it, vi } from "vitest"

import { RequireSession } from "@/features/auth/auth-provider"
import { OrgWorkspaceLayout } from "@/features/orgs/layouts/org-workspace-layout"
import type { Session } from "@/features/auth/session"
import { LinksPage } from "@/features/links/pages/links-page"
import type { ShortLink } from "@/features/links/types"
import { AuthTestProvider } from "@/test/auth"

const session: Session = {
  userId: "user-1",
  email: "ada@example.com",
  orgs: { "org-1": ["*"] },
}

const org = {
  id: "org-1",
  name: "Acme",
  userId: "user-1",
  tags: ["team"],
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
}

const launch: ShortLink = {
  id: "link-1",
  orgId: "org-1",
  userId: "user-1",
  slug: "launch",
  url: "https://example.com/page",
  title: "Launch page",
  tags: ["marketing", "launch"],
  shortPath: "/s/launch",
  shortUrl: "https://api.example.com/s/launch",
  disabled: false,
  createdAt: "2026-11-20T10:00:00Z",
  updatedAt: "2026-11-20T10:00:00Z",
}

const docs: ShortLink = {
  ...launch,
  id: "link-2",
  slug: "docs",
  url: "https://docs.example.com",
  title: "Docs",
  tags: ["docs"],
  shortPath: "/s/docs",
  shortUrl: "https://api.example.com/s/docs",
  disabled: true,
}

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => (body === undefined ? "" : JSON.stringify(body)),
  } as Response
}

function stubApi(initial: ShortLink[] = []) {
  const links = [...initial]

  const fetchMock = vi.fn<typeof fetch>(async (input, init) => {
    const url = new URL(String(input), "http://localhost")
    const method = init?.method ?? "GET"

    if (method === "GET" && url.pathname === "/api/orgs/org-1") {
      return jsonResponse(org)
    }

    if (method === "GET" && url.pathname === "/api/orgs/org-1/links") {
      const query = (url.searchParams.get("q") ?? "").toLowerCase()
      const tags = (url.searchParams.get("tags") ?? "")
        .split(",")
        .filter((tag) => tag !== "")
      const limit = Number(url.searchParams.get("limit") ?? "20")
      const offset = Number(url.searchParams.get("offset") ?? "0")

      let filtered = links
      if (query !== "") {
        filtered = filtered.filter((link) =>
          link.slug.toLowerCase().startsWith(query)
        )
      }
      if (tags.length > 0) {
        filtered = filtered.filter((link) =>
          link.tags.some((tag) => tags.includes(tag))
        )
      }

      const items = filtered.slice(offset, offset + limit)
      const nextOffset =
        offset + limit < filtered.length ? offset + limit : null
      return jsonResponse({ items, nextOffset })
    }

    if (method === "POST" && url.pathname === "/api/orgs/org-1/links") {
      const body = JSON.parse(String(init?.body)) as {
        url: string
        slug?: string
        title?: string
        tags?: string[]
      }
      const slug = body.slug ?? "random1"
      const created: ShortLink = {
        ...launch,
        id: `link-created-${links.length + 1}`,
        slug,
        url: body.url,
        title: body.title ?? "",
        tags: body.tags ?? [],
        shortPath: `/s/${slug}`,
        shortUrl: `https://api.example.com/s/${slug}`,
        disabled: false,
      }
      links.unshift(created)
      return jsonResponse(created, 201)
    }

    if (
      method === "PATCH" &&
      url.pathname.startsWith("/api/orgs/org-1/links/")
    ) {
      const id = url.pathname.split("/").pop() ?? ""
      const link = links.find((item) => item.id === id)
      if (!link) {
        return jsonResponse({ error: "link not found" }, 404)
      }
      const body = JSON.parse(String(init?.body)) as Partial<ShortLink>
      Object.assign(link, body)
      return jsonResponse(link)
    }

    if (
      method === "DELETE" &&
      url.pathname.startsWith("/api/orgs/org-1/links/")
    ) {
      const id = url.pathname.split("/").pop() ?? ""
      const index = links.findIndex((item) => item.id === id)
      if (index >= 0) {
        links.splice(index, 1)
      }
      return jsonResponse(undefined, 204)
    }

    return jsonResponse({ error: "not found" }, 404)
  })

  vi.stubGlobal("fetch", fetchMock)
  return { fetchMock, links }
}

function renderLinksPage(
  initialSession: Session | null = session,
  initialEntry = "/orgs/org-1/links"
) {
  return render(
    <AuthTestProvider session={initialSession}>
      <MemoryRouter initialEntries={[initialEntry]}>
        <Routes>
          <Route element={<RequireSession />}>
            <Route path="/orgs/:orgId" element={<OrgWorkspaceLayout />}>
              <Route path="links" element={<LinksPage />} />
            </Route>
          </Route>
          <Route
            path="/orgs/:orgId"
            element={<div>Organization destination</div>}
          />
          <Route path="/login" element={<div>Login destination</div>} />
        </Routes>
      </MemoryRouter>
    </AuthTestProvider>
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe("LinksPage", () => {
  it("loads the organization links", async () => {
    stubApi([launch, docs])

    renderLinksPage()

    expect(
      await screen.findByRole("heading", { name: "Links" })
    ).toBeInTheDocument()
    expect(await screen.findByText("Launch page")).toBeInTheDocument()
    expect(screen.getByText("/s/launch")).toBeInTheDocument()
    expect(screen.getByText("Active")).toBeInTheDocument()
    expect(screen.getByText("Disabled")).toBeInTheDocument()
    expect(screen.getByText("marketing")).toBeInTheDocument()
  })

  it("creates a link with a custom alias", async () => {
    const { fetchMock } = stubApi([])
    const user = userEvent.setup()

    renderLinksPage()
    await screen.findByText("No links yet")

    await user.click(screen.getByRole("button", { name: "New link" }))
    await screen.findByRole("dialog", { name: "Create link" })

    await user.type(
      screen.getByLabelText("Destination URL"),
      "https://example.com/launch"
    )
    await user.type(screen.getByLabelText("Custom alias"), "launch")
    await user.type(screen.getByLabelText("Title"), "Launch page")
    await user.click(screen.getByRole("button", { name: "Create link" }))

    expect(await screen.findByText("Launch page")).toBeInTheDocument()

    const post = fetchMock.mock.calls.find(
      ([url, init]) =>
        String(url).startsWith("/api/orgs/org-1/links?") &&
        init?.method === "POST"
    )
    expect(post).toBeDefined()
    expect(JSON.parse(String(post?.[1]?.body))).toMatchObject({
      url: "https://example.com/launch",
      slug: "launch",
      title: "Launch page",
    })
  })

  it("edits the destination", async () => {
    const { fetchMock } = stubApi([launch])
    const user = userEvent.setup()

    renderLinksPage()
    await screen.findByText("Launch page")

    await user.click(screen.getByRole("button", { name: "Edit launch" }))
    await screen.findByRole("dialog", { name: "Edit link" })
    const dialog = screen.getByRole("dialog", { name: "Edit link" })
    const urlInput = within(dialog).getByLabelText("Destination URL")
    await user.clear(urlInput)
    await user.type(urlInput, "https://example.com/new")
    await user.click(
      within(dialog).getByRole("button", { name: "Save changes" })
    )

    await waitFor(() => {
      expect(
        screen.queryByRole("dialog", { name: "Edit link" })
      ).not.toBeInTheDocument()
    })
    const patch = fetchMock.mock.calls.find(
      ([, init]) => init?.method === "PATCH"
    )
    expect(JSON.parse(String(patch?.[1]?.body))).toMatchObject({
      url: "https://example.com/new",
    })
  })

  it("disables a link", async () => {
    const { fetchMock } = stubApi([launch])
    const user = userEvent.setup()

    renderLinksPage()
    await screen.findByText("Launch page")

    await user.click(screen.getByRole("button", { name: "Disable launch" }))

    expect(await screen.findByText("Disabled")).toBeInTheDocument()
    const patch = fetchMock.mock.calls.find(
      ([, init]) => init?.method === "PATCH"
    )
    expect(JSON.parse(String(patch?.[1]?.body))).toEqual({ disabled: true })
  })

  it("deletes a link after confirmation", async () => {
    const { fetchMock } = stubApi([launch])
    const user = userEvent.setup()

    renderLinksPage()
    await screen.findByText("Launch page")

    await user.click(screen.getByRole("button", { name: "Delete launch" }))
    await screen.findByRole("alertdialog", { name: "Delete link?" })
    await user.click(screen.getByRole("button", { name: "Delete link" }))

    expect(await screen.findByText("No links yet")).toBeInTheDocument()
    expect(
      fetchMock.mock.calls.some(([, init]) => init?.method === "DELETE")
    ).toBe(true)
  })

  it("searches by alias prefix", async () => {
    const { fetchMock } = stubApi([launch, docs])
    const user = userEvent.setup()

    renderLinksPage()
    await screen.findByText("Launch page")

    await user.type(screen.getByLabelText("Search aliases"), "do")

    await waitFor(() => {
      expect(
        fetchMock.mock.calls.some(
          ([url]) =>
            String(url).includes("/api/orgs/org-1/links?") &&
            String(url).includes("q=do")
        )
      ).toBe(true)
    })
    await waitFor(() => {
      expect(screen.queryByText("Launch page")).not.toBeInTheDocument()
    })
    expect(screen.getByText("Docs")).toBeInTheDocument()
  })

  it("filters by tags", async () => {
    const { fetchMock } = stubApi([launch, docs])
    const user = userEvent.setup()

    renderLinksPage()
    await screen.findByText("Launch page")

    await user.type(screen.getByLabelText("Filter tags (any)"), "docs{Enter}")

    await waitFor(() => {
      expect(
        fetchMock.mock.calls.some(
          ([url]) =>
            String(url).includes("/api/orgs/org-1/links?") &&
            String(url).includes("tags=docs")
        )
      ).toBe(true)
    })
    await waitFor(() => {
      expect(screen.queryByText("Launch page")).not.toBeInTheDocument()
    })
    expect(screen.getByText("Docs")).toBeInTheDocument()
  })

  it("opens the create dialog from the new query parameter", async () => {
    stubApi([])

    renderLinksPage(session, "/orgs/org-1/links?new=1")

    expect(
      await screen.findByRole("dialog", { name: "Create link" })
    ).toBeInTheDocument()
  })

  it("shows a forbidden state without a link role", async () => {
    const fetchMock = vi.fn<typeof fetch>(async (input) => {
      const url = new URL(String(input), "http://localhost")
      if (url.pathname === "/api/orgs/org-1") {
        return jsonResponse(org)
      }
      return jsonResponse({ error: "unexpected request" }, 500)
    })
    vi.stubGlobal("fetch", fetchMock)

    renderLinksPage({
      userId: "user-2",
      email: "grace@example.com",
      orgs: { "org-1": ["project.viewer"] },
    })

    expect(
      await screen.findByText(
        "You do not have access to links in this organization."
      )
    ).toBeInTheDocument()
    expect(
      fetchMock.mock.calls.some(([url]) =>
        String(url).startsWith("/api/orgs/org-1/links")
      )
    ).toBe(false)
  })
})
