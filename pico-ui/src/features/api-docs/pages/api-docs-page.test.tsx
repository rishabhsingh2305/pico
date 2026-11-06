import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter } from "react-router"
import { afterEach, describe, expect, it, vi } from "vitest"

import { ApiDocsPage } from "@/features/api-docs/pages/api-docs-page"
import { AuthTestProvider } from "@/test/auth"

function renderDocs() {
  return render(
    <AuthTestProvider session={null}>
      <MemoryRouter initialEntries={["/api-docs"]}>
        <ApiDocsPage />
      </MemoryRouter>
    </AuthTestProvider>
  )
}

afterEach(() => {
  Object.defineProperty(navigator, "clipboard", {
    configurable: true,
    value: undefined,
  })
})

describe("ApiDocsPage", () => {
  it("renders every API section without a session", () => {
    renderDocs()

    expect(
      screen.getByRole("heading", { name: "Pico API" })
    ).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "Sign in" })).toHaveAttribute(
      "href",
      "/login"
    )

    for (const section of [
      "Health",
      "Authentication",
      "Users",
      "Email verification",
      "Organizations",
      "Memberships",
      "Projects",
      "Expenses",
    ]) {
      expect(screen.getByRole("heading", { name: section })).toBeInTheDocument()
    }
  })

  it("filters endpoints by search and clears the search", async () => {
    const user = userEvent.setup()
    renderDocs()

    const search = screen.getByLabelText("Search endpoints")
    await user.type(search, "change password")

    expect(
      screen.getByRole("heading", { name: "Change password" })
    ).toBeInTheDocument()
    expect(
      screen.queryByRole("heading", { name: "Check liveness" })
    ).not.toBeInTheDocument()
    expect(screen.getByText("1 matching endpoint")).toBeInTheDocument()

    await user.click(
      screen.getByRole("button", { name: "Clear endpoint search" })
    )

    expect(
      screen.getByRole("heading", { name: "Check liveness" })
    ).toBeInTheDocument()
  })

  it("shows an empty state when nothing matches", async () => {
    const user = userEvent.setup()
    renderDocs()

    await user.type(screen.getByLabelText("Search endpoints"), "zzzz")

    expect(
      screen.getByText("No endpoints match your search")
    ).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "Clear search" }))

    expect(
      screen.getByRole("heading", { name: "Check liveness" })
    ).toBeInTheDocument()
  })

  it("copies endpoint paths and shows feedback", async () => {
    const user = userEvent.setup()
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    })
    renderDocs()

    await user.click(screen.getByRole("button", { name: "Copy GET /ping" }))

    expect(writeText).toHaveBeenCalledWith("/ping")
    expect(screen.getByText("Copied")).toBeInTheDocument()
  })
})
