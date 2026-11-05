import { useState } from "react"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it } from "vitest"

import { DateRangePicker } from "@/components/date-range-picker"
import type { DateRangeValue } from "@/lib/date-range"

const today = new Date(2026, 10, 5)

function Harness({ initial }: { initial?: DateRangeValue }) {
  const [range, setRange] = useState<DateRangeValue>(
    initial ?? { from: "", to: "" }
  )

  return (
    <div>
      <DateRangePicker
        from={range.from}
        to={range.to}
        onChange={setRange}
        today={today}
      />
      <output data-testid="range">
        {range.from}|{range.to}
      </output>
    </div>
  )
}

describe("DateRangePicker", () => {
  it("applies a preset and shows its label", async () => {
    const user = userEvent.setup()
    render(<Harness />)

    expect(
      screen.getByRole("button", { name: "Date range: Any date" })
    ).toBeInTheDocument()

    await user.click(
      screen.getByRole("button", { name: "Date range: Any date" })
    )
    await user.click(screen.getByRole("button", { name: "Last 7 days" }))

    expect(screen.getByTestId("range")).toHaveTextContent(
      "2026-10-30|2026-11-05"
    )
    expect(
      screen.getByRole("button", { name: "Date range: Last 7 days" })
    ).toBeInTheDocument()
  })

  it("selects a custom start and end date", async () => {
    const user = userEvent.setup()
    render(<Harness />)

    await user.click(
      screen.getByRole("button", { name: "Date range: Any date" })
    )

    const start = document.querySelector<HTMLButtonElement>(
      '[data-day="2026-11-02"] button'
    )
    expect(start).not.toBeNull()

    await user.click(start as HTMLButtonElement)
    expect(screen.getByTestId("range")).toHaveTextContent("|")

    const end = document.querySelector<HTMLButtonElement>(
      '[data-day="2026-11-05"] button'
    )
    expect(end).not.toBeNull()
    await user.click(end as HTMLButtonElement)

    expect(screen.getByTestId("range")).toHaveTextContent(
      "2026-11-02|2026-11-05"
    )
    expect(
      screen.getByRole("button", { name: "Date range: Nov 2 – Nov 5" })
    ).toBeInTheDocument()
  })

  it("clears an applied range", async () => {
    const user = userEvent.setup()
    render(<Harness initial={{ from: "2026-10-30", to: "2026-11-05" }} />)

    await user.click(
      screen.getByRole("button", { name: "Date range: Last 7 days" })
    )
    await user.click(screen.getByRole("button", { name: "Clear dates" }))

    expect(screen.getByTestId("range")).toHaveTextContent("|")
    expect(
      screen.getByRole("button", { name: "Date range: Any date" })
    ).toBeInTheDocument()
  })

  it("shows a partial range when only a start is known", () => {
    render(<Harness initial={{ from: "2026-11-02", to: "" }} />)

    expect(
      screen.getByRole("button", { name: "Date range: From Nov 2, 2026" })
    ).toBeInTheDocument()
  })
})
