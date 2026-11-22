import { render, screen } from "@testing-library/react"
import { MemoryRouter } from "react-router"
import { describe, expect, it } from "vitest"

import { AnalyticsList } from "@/features/analytics/components/analytics-list"
import type { AnalyticsSource } from "@/features/analytics/types"

const active: AnalyticsSource = {
  id: "source-1",
  orgId: "org-1",
  name: "Homepage CTA",
  description: "Top hero button",
  key: "abc123",
  events: ["view", "click"],
  pixelPath: "/p/abc123",
  pixelUrl: "https://link.example.com/p/abc123",
  disabled: false,
  createdAt: "2026-11-20T10:00:00Z",
  updatedAt: "2026-11-20T10:00:00Z",
}

const disabled: AnalyticsSource = {
  ...active,
  id: "source-2",
  name: "Email footer",
  description: "",
  key: "def456",
  pixelPath: "/p/def456",
  pixelUrl: "https://link.example.com/p/def456",
  disabled: true,
}

function renderList(
  captures: ReadonlyMap<string, number>,
  statsLoading = false,
  statsError = false
) {
  return render(
    <MemoryRouter>
      <AnalyticsList
        orgId="org-1"
        sources={[active, disabled]}
        capturesBySourceId={captures}
        statsLoading={statsLoading}
        statsError={statsError}
      />
    </MemoryRouter>
  )
}

describe("AnalyticsList", () => {
  it("renders sources, pixel paths, statuses, and capture totals", () => {
    renderList(new Map([["source-1", 1234]]))

    expect(screen.getByText("Homepage CTA")).toBeInTheDocument()
    expect(screen.getByText("Top hero button")).toBeInTheDocument()
    expect(screen.getByText("/p/abc123")).toBeInTheDocument()
    expect(screen.getByText("Email footer")).toBeInTheDocument()
    expect(screen.getByText("/p/def456")).toBeInTheDocument()
    expect(screen.getByText("Active")).toBeInTheDocument()
    expect(screen.getByText("Disabled")).toBeInTheDocument()
    expect(screen.getByText("1,234")).toBeInTheDocument()
    expect(screen.getByText("0")).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "Homepage CTA" })).toHaveAttribute(
      "href",
      "/orgs/org-1/analytics/source-1"
    )
  })

  it("shows a placeholder while totals load", () => {
    renderList(new Map(), true)
    expect(screen.getAllByLabelText("Loading captures")).toHaveLength(2)
  })

  it("shows a placeholder when totals fail", () => {
    renderList(new Map(), false, true)
    expect(screen.getAllByLabelText("Captures unavailable")).toHaveLength(2)
  })

  it("renders nothing without sources", () => {
    const { container } = render(
      <MemoryRouter>
        <AnalyticsList
          orgId="org-1"
          sources={[]}
          capturesBySourceId={new Map()}
          statsLoading={false}
          statsError={false}
        />
      </MemoryRouter>
    )
    expect(container).toBeEmptyDOMElement()
  })
})
