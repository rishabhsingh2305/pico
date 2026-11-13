import { render, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter, Route, Routes } from "react-router"
import { afterEach, describe, expect, it, vi } from "vitest"

import type { Session } from "@/features/auth/session"
import { RequireSession } from "@/features/auth/auth-provider"
import { OrgDetailPage } from "@/features/orgs/pages/org-detail-page"
import type { Org } from "@/features/orgs/types"
import type { Project } from "@/features/projects/types"
import { AuthTestProvider } from "@/test/auth"

const session: Session = {
  userId: "user-1",
  email: "ada@example.com",
}

const org: Org = {
  id: "org-1",
  name: "Acme",
  userId: "user-1",
  tags: ["ops"],
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
}

const general: Project = {
  id: "project-1",
  orgId: "org-1",
  userId: "user-1",
  name: "General",
  currency: "USD",
  tags: ["travel"],
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
}

const foodProject: Project = {
  ...general,
  id: "project-food",
  name: "Food",
  tags: ["food"],
}

const mixedProject: Project = {
  ...general,
  id: "project-mixed",
  name: "Mixed",
  tags: ["food", "travel"],
}

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => (body === undefined ? "" : JSON.stringify(body)),
  } as Response
}

type ApiOptions = {
  orgResponse?: Response
  projects?: Project[]
  projectsResponse?: Response
}

function stubApi(options: ApiOptions = {}) {
  const projects = [...(options.projects ?? [])]

  const fetchMock = vi.fn<typeof fetch>(async (input, init) => {
    const url = new URL(String(input), "http://localhost")
    const method = init?.method ?? "GET"

    if (method === "GET" && url.pathname === "/api/orgs/org-1") {
      return options.orgResponse ?? jsonResponse(org)
    }
    if (method === "GET" && url.pathname === "/api/projects") {
      if (options.projectsResponse) {
        return options.projectsResponse
      }
      const tags = (url.searchParams.get("tags") ?? "")
        .split(",")
        .filter((tag) => tag !== "")
      const filtered =
        tags.length === 0
          ? projects
          : projects.filter((project) =>
              project.tags.some((tag) => tags.includes(tag))
            )
      return jsonResponse({ items: filtered })
    }
    if (method === "POST" && url.pathname === "/api/projects") {
      const body = JSON.parse(String(init?.body)) as {
        name: string
        currency: string
        tags?: string[]
      }
      const created: Project = {
        ...general,
        id: `project-created-${projects.length + 1}`,
        name: body.name,
        currency: body.currency,
        tags: body.tags ?? [],
      }
      projects.unshift(created)
      return jsonResponse(created, 201)
    }
    if (method === "PATCH" && url.pathname.startsWith("/api/projects/")) {
      const id = url.pathname.split("/").pop() ?? ""
      const body = JSON.parse(String(init?.body)) as { tags?: string[] }
      const project = projects.find((item) => item.id === id)
      if (!project) {
        return jsonResponse({ error: "project not found" }, 404)
      }
      if (body.tags) {
        project.tags = body.tags
      }
      return jsonResponse(project)
    }

    return jsonResponse({ error: "not found" }, 404)
  })

  vi.stubGlobal("fetch", fetchMock)
  return fetchMock
}

function projectListURLs(fetchMock: ReturnType<typeof stubApi>): URL[] {
  return fetchMock.mock.calls
    .filter(
      ([url, init]) =>
        String(url).startsWith("/api/projects?") &&
        (init?.method ?? "GET") === "GET"
    )
    .map(([url]) => new URL(String(url), "http://localhost"))
}

function renderOrgDetail(initialSession: Session | null = session) {
  return render(
    <AuthTestProvider session={initialSession}>
      <MemoryRouter initialEntries={["/orgs/org-1"]}>
        <Routes>
          <Route element={<RequireSession />}>
            <Route path="/orgs/:orgId" element={<OrgDetailPage />} />
          </Route>
          <Route path="/orgs" element={<div>Organizations destination</div>} />
          <Route path="/login" element={<div>Login destination</div>} />
        </Routes>
      </MemoryRouter>
    </AuthTestProvider>
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe("OrgDetailPage", () => {
  it("redirects to login without a session", () => {
    renderOrgDetail(null)

    expect(screen.getByText("Login destination")).toBeInTheDocument()
  })

  it("loads the organization, projects, and project tags", async () => {
    const fetchMock = stubApi({ projects: [general] })

    renderOrgDetail()

    expect(
      await screen.findByRole("heading", { name: "Acme" })
    ).toBeInTheDocument()
    expect(screen.getByText("ops")).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "Dashboard" })).toHaveAttribute(
      "href",
      "/orgs/org-1/dashboard"
    )
    expect(
      await screen.findByRole("link", { name: "General" })
    ).toHaveAttribute("href", "/orgs/org-1/projects/project-1")
    expect(
      screen.getByRole("button", { name: "Filter by travel" })
    ).toBeInTheDocument()
    expect(screen.queryByText("Team lunch")).not.toBeInTheDocument()

    const projectURL = projectListURLs(fetchMock)[0]
    expect(projectURL.searchParams.get("orgId")).toBe("org-1")
    expect(projectURL.searchParams.get("userId")).toBe("user-1")
  })

  it("opens project creation in a modal when none exist", async () => {
    stubApi({ projects: [] })
    const userEventInstance = userEvent.setup()

    renderOrgDetail()

    expect(await screen.findByText("No projects yet")).toBeInTheDocument()
    expect(screen.queryByLabelText("Project name")).not.toBeInTheDocument()

    await userEventInstance.click(
      screen.getByRole("button", { name: "New project" })
    )

    expect(
      await screen.findByRole("dialog", { name: "Create project" })
    ).toBeInTheDocument()
    expect(screen.getByLabelText("Project name")).toBeInTheDocument()
    expect(screen.getByLabelText("Tags")).toBeInTheDocument()
  })

  it("closes project creation without submitting on cancel", async () => {
    const fetchMock = stubApi({ projects: [] })
    const userEventInstance = userEvent.setup()

    renderOrgDetail()
    await screen.findByText("No projects yet")

    await userEventInstance.click(
      screen.getByRole("button", { name: "New project" })
    )
    await screen.findByRole("dialog", { name: "Create project" })
    await userEventInstance.click(
      screen.getByRole("button", { name: "Cancel" })
    )

    await waitFor(() => {
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
    })
    expect(
      fetchMock.mock.calls.some(([, init]) => init?.method === "POST")
    ).toBe(false)
  })

  it("creates a project with tags and adds it to the list", async () => {
    const fetchMock = stubApi({ projects: [] })
    const userEventInstance = userEvent.setup()

    renderOrgDetail()
    await screen.findByText("No projects yet")
    await userEventInstance.click(
      screen.getByRole("button", { name: "New project" })
    )
    await screen.findByRole("dialog", { name: "Create project" })

    await userEventInstance.type(screen.getByLabelText("Project name"), "Trips")
    await userEventInstance.type(screen.getByLabelText("Tags"), "travel,")
    await userEventInstance.click(
      screen.getByRole("button", { name: "Create project" })
    )

    expect(await screen.findByRole("link", { name: "Trips" })).toHaveAttribute(
      "href",
      "/orgs/org-1/projects/project-created-1"
    )
    expect(
      screen.getByRole("button", { name: "Filter by travel" })
    ).toBeInTheDocument()
    await waitFor(() => {
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
    })

    const createProjectCall = fetchMock.mock.calls.find(
      ([url, init]) =>
        url === "/api/projects?orgId=org-1&userId=user-1" &&
        init?.method === "POST"
    )
    expect(createProjectCall).toBeDefined()
    expect(JSON.parse(String(createProjectCall?.[1]?.body))).toEqual({
      name: "Trips",
      currency: "USD",
      tags: ["travel"],
    })
  })

  it("filters projects server-side with any-match tags", async () => {
    const fetchMock = stubApi({
      projects: [general, foodProject, mixedProject],
    })
    const userEventInstance = userEvent.setup()

    renderOrgDetail()
    await screen.findByRole("link", { name: "General" })

    const filterForm = screen.getByLabelText("Filter tags (any)")
    await userEventInstance.type(filterForm, "food,")

    await waitFor(() => {
      expect(projectListURLs(fetchMock).at(-1)?.searchParams.get("tags")).toBe(
        "food"
      )
    })
    expect(
      await screen.findByRole("link", { name: "Food" })
    ).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "Mixed" })).toBeInTheDocument()
    expect(
      screen.queryByRole("link", { name: "General" })
    ).not.toBeInTheDocument()

    await userEventInstance.type(
      screen.getByLabelText("Filter tags (any)"),
      "travel,"
    )

    await waitFor(() => {
      expect(projectListURLs(fetchMock).at(-1)?.searchParams.get("tags")).toBe(
        "food,travel"
      )
    })
    expect(
      await screen.findByRole("link", { name: "General" })
    ).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "Food" })).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "Mixed" })).toBeInTheDocument()
  })

  it("toggles a project tag from the card into the filter", async () => {
    const fetchMock = stubApi({
      projects: [general, foodProject],
    })
    const userEventInstance = userEvent.setup()

    renderOrgDetail()
    await screen.findByRole("link", { name: "General" })

    await userEventInstance.click(
      screen.getByRole("button", { name: "Filter by travel" })
    )

    await waitFor(() => {
      expect(projectListURLs(fetchMock).at(-1)?.searchParams.get("tags")).toBe(
        "travel"
      )
    })
    expect(
      await screen.findByRole("button", { name: "Filter by travel" })
    ).toHaveAttribute("aria-pressed", "true")
    expect(screen.queryByRole("link", { name: "Food" })).not.toBeInTheDocument()

    await userEventInstance.click(
      screen.getByRole("button", { name: "Filter by travel" })
    )

    expect(
      await screen.findByRole("link", { name: "Food" })
    ).toBeInTheDocument()
    expect(
      projectListURLs(fetchMock).some(
        (url) => url.searchParams.get("tags") === null
      )
    ).toBe(true)
  })

  it("shows a filtered empty state and clears the filter", async () => {
    stubApi({ projects: [general] })
    const userEventInstance = userEvent.setup()

    renderOrgDetail()
    await screen.findByRole("link", { name: "General" })

    await userEventInstance.type(
      screen.getByLabelText("Filter tags (any)"),
      "missing,"
    )

    expect(
      await screen.findByText("No projects match these tags")
    ).toBeInTheDocument()
    expect(
      screen.queryByRole("link", { name: "General" })
    ).not.toBeInTheDocument()

    await userEventInstance.click(screen.getByRole("button", { name: "Clear" }))

    expect(
      await screen.findByRole("link", { name: "General" })
    ).toBeInTheDocument()
  })

  it("edits project tags and refreshes the list", async () => {
    const fetchMock = stubApi({ projects: [general] })
    const userEventInstance = userEvent.setup()

    renderOrgDetail()
    await screen.findByRole("link", { name: "General" })

    await userEventInstance.click(
      screen.getByRole("button", { name: "Edit tags for General" })
    )
    const dialog = await screen.findByRole("dialog", {
      name: "Edit project tags",
    })
    expect(within(dialog).getByText("travel")).toBeInTheDocument()

    await userEventInstance.type(within(dialog).getByLabelText("Tags"), "food,")
    await userEventInstance.click(
      within(dialog).getByRole("button", { name: "Save tags" })
    )

    await waitFor(() => {
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
    })
    expect(
      await screen.findByRole("button", { name: "Filter by food" })
    ).toBeInTheDocument()

    const updateCall = fetchMock.mock.calls.find(
      ([url, init]) =>
        String(url).startsWith("/api/projects/project-1?") &&
        init?.method === "PATCH"
    )
    expect(updateCall).toBeDefined()
    expect(JSON.parse(String(updateCall?.[1]?.body))).toEqual({
      tags: ["travel", "food"],
    })
  })

  it("shows a project list error with a retry action", async () => {
    stubApi({
      projectsResponse: jsonResponse({ error: "list failed" }, 500),
    })

    renderOrgDetail()

    expect(
      await screen.findByText("Something went wrong. Please try again.")
    ).toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: "Try again" })
    ).toBeInTheDocument()
  })

  it("shows organization-not-found with a back action", async () => {
    stubApi({
      orgResponse: jsonResponse({ error: "org not found" }, 404),
    })

    renderOrgDetail()

    expect(
      await screen.findByText("Organization not found.")
    ).toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: "Back to organizations" })
    ).toBeInTheDocument()
  })

  it("clears a stale user session", async () => {
    stubApi({
      orgResponse: jsonResponse({ error: "user not found" }, 404),
    })

    renderOrgDetail()

    expect(await screen.findByText("Login destination")).toBeInTheDocument()
  })
})
