import { render, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter, Route, Routes } from "react-router"
import { afterEach, describe, expect, it, vi } from "vitest"

import { RequireSession } from "@/features/auth/auth-provider"
import type { Session } from "@/features/auth/session"
import { OrgWorkspaceLayout } from "@/features/orgs/layouts/org-workspace-layout"
import { SharingPage } from "@/features/sharing/pages/sharing-page"
import type { ShareCredential } from "@/features/sharing/types"
import { AuthTestProvider } from "@/test/auth"

const session: Session = {
  userId: "user-1",
  email: "ada@example.com",
}

const org = {
  id: "org-1",
  name: "Acme",
  userId: "user-1",
  tags: [],
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
}

const project = {
  id: "project-1",
  orgId: "org-1",
  userId: "user-1",
  name: "Launch",
  currency: "INR",
  tags: [],
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
}

const share: ShareCredential = {
  id: "share-1",
  kind: "share",
  orgId: "org-1",
  resourceType: "project",
  resourceId: "project-1",
  projectId: "project-1",
  scopes: ["orgs.get", "projects.get", "expenses.list"],
  name: "Client review",
  createdBy: "user-1",
  createdAt: "2026-11-20T10:00:00Z",
  expiresAt: "2026-12-20T10:00:00Z",
  status: "active",
}

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  } as Response
}

function stubApi() {
  const fetchMock = vi.fn<typeof fetch>(async (input, init) => {
    const url = new URL(String(input), "http://localhost")
    const method = init?.method ?? "GET"

    if (method === "GET" && url.pathname === "/api/orgs/org-1") {
      return jsonResponse(org)
    }
    if (method === "GET" && url.pathname === "/api/projects") {
      return jsonResponse({ items: [project] })
    }
    if (method === "GET" && url.pathname === "/api/orgs/org-1/credentials") {
      return jsonResponse({ items: [share] })
    }
    if (method === "POST" && url.pathname === "/api/orgs/org-1/shares") {
      const body = JSON.parse(String(init?.body)) as {
        resourceType: "org" | "project"
        resourceId?: string
        name?: string
      }
      return jsonResponse(
        {
          ...share,
          id: "share-created",
          resourceType: body.resourceType,
          resourceId: body.resourceId ?? "org-1",
          projectId:
            body.resourceType === "project" ? body.resourceId : undefined,
          name: body.name,
          token: "pico_share_test",
          sharePath: "/shared#share=pico_share_test",
        },
        201
      )
    }
    if (
      method === "DELETE" &&
      url.pathname === "/api/orgs/org-1/shares/share-1"
    ) {
      return jsonResponse(undefined, 204)
    }
    return jsonResponse({ error: "not found" }, 404)
  })
  vi.stubGlobal("fetch", fetchMock)
  return fetchMock
}

function renderSharing(initialSession: Session | null = session) {
  return render(
    <AuthTestProvider session={initialSession}>
      <MemoryRouter initialEntries={["/orgs/org-1/sharing"]}>
        <Routes>
          <Route element={<RequireSession />}>
            <Route path="/orgs/:orgId" element={<OrgWorkspaceLayout />}>
              <Route path="sharing" element={<SharingPage />} />
            </Route>
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

describe("SharingPage", () => {
  it("lists share links with their permissions", async () => {
    stubApi()

    renderSharing()

    expect(
      await screen.findByRole("heading", { name: "Sharing" })
    ).toBeInTheDocument()
    const row = await screen.findByRole("button", { name: /Client review/ })
    expect(row).toBeInTheDocument()
    await userEvent.click(row)

    const details = await screen.findByRole("dialog", {
      name: "Client review",
    })
    expect(within(details).getByText(/Project · Launch/)).toBeInTheDocument()
    expect(within(details).getByText("Read organization")).toBeInTheDocument()
    expect(within(details).getByText("Read project")).toBeInTheDocument()
    expect(within(details).getByText("List expenses")).toBeInTheDocument()
    expect(within(details).getAllByText("Active").length).toBeGreaterThan(0)
  })

  it("creates a project share and shows the link once", async () => {
    const fetchMock = stubApi()
    const user = userEvent.setup()

    renderSharing()
    await screen.findByText("Client review")

    await user.click(screen.getByRole("button", { name: "New share link" }))
    const dialog = await screen.findByRole("dialog", {
      name: "Create share link",
    })

    await user.click(within(dialog).getByLabelText("Resource"))
    await user.click(await screen.findByRole("option", { name: "Project" }))
    await user.click(within(dialog).getByLabelText("Project"))
    await user.click(await screen.findByRole("option", { name: "Launch" }))
    await user.type(within(dialog).getByLabelText("Name"), "QA review")
    await user.click(
      within(dialog).getByRole("button", { name: "Create link" })
    )

    const link = (await screen.findByLabelText(
      "Share link"
    )) as HTMLInputElement
    expect(link.value).toContain("pico_share_test")

    const post = fetchMock.mock.calls.find(
      ([url, init]) =>
        String(url).startsWith("/api/orgs/org-1/shares") &&
        init?.method === "POST"
    )
    expect(post).toBeDefined()
    expect(JSON.parse(String(post?.[1]?.body))).toMatchObject({
      resourceType: "project",
      resourceId: "project-1",
      name: "QA review",
    })
  })

  it("revokes an active share link", async () => {
    const fetchMock = stubApi()
    const user = userEvent.setup()

    renderSharing()
    const row = await screen.findByRole("button", { name: /Client review/ })
    await user.click(row)
    const details = await screen.findByRole("dialog", {
      name: "Client review",
    })

    await user.click(
      within(details).getByRole("button", { name: "Revoke link" })
    )
    const dialog = await screen.findByRole("alertdialog", {
      name: "Revoke share link?",
    })
    await user.click(
      within(dialog).getByRole("button", { name: "Revoke link" })
    )

    await waitFor(() => {
      expect(
        fetchMock.mock.calls.some(
          ([url, init]) =>
            String(url).startsWith("/api/orgs/org-1/shares/share-1") &&
            init?.method === "DELETE"
        )
      ).toBe(true)
    })
  })
})
