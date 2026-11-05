import { StrictMode } from "react"
import { render, screen, waitFor } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import { CountUp } from "@/components/count-up"

describe("CountUp", () => {
  it("renders zero and counts up to the final value", async () => {
    render(
      <StrictMode>
        <CountUp
          value={1299}
          format={(value) => value.toString()}
          durationMs={20}
        />
      </StrictMode>
    )

    expect(screen.getByText("0")).toBeInTheDocument()

    await waitFor(() => {
      expect(screen.getByText("1299")).toBeInTheDocument()
    })
  })

  it("counts down to a negative net value", async () => {
    render(
      <StrictMode>
        <CountUp
          value={-1299}
          format={(value) => value.toString()}
          durationMs={20}
        />
      </StrictMode>
    )

    expect(screen.getByText("0")).toBeInTheDocument()

    await waitFor(() => {
      expect(screen.getByText("-1299")).toBeInTheDocument()
    })
  })
})
