import { useState } from "react"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it } from "vitest"

import { TagInput } from "@/components/tag-input"

function Harness({ initialTags = [] }: { initialTags?: string[] }) {
  const [tags, setTags] = useState(initialTags)

  return (
    <div>
      <label htmlFor="tags">Tags</label>
      <TagInput id="tags" name="tags" value={tags} onChange={setTags} />
      <p data-testid="committed-tags">{JSON.stringify(tags)}</p>
    </div>
  )
}

describe("TagInput", () => {
  it("turns comma-separated text into badges", async () => {
    const user = userEvent.setup()
    render(<Harness />)

    const input = screen.getByLabelText("Tags")
    await user.type(input, "food,")

    expect(screen.getByText("food")).toBeInTheDocument()
    expect(input).toHaveValue("")
    expect(screen.getByTestId("committed-tags")).toHaveTextContent(
      JSON.stringify(["food"])
    )
  })

  it("commits a pending draft on blur", async () => {
    const user = userEvent.setup()
    render(<Harness />)

    await user.type(screen.getByLabelText("Tags"), "food")
    await user.tab()

    expect(screen.getByText("food")).toBeInTheDocument()
    expect(screen.getByTestId("committed-tags")).toHaveTextContent(
      JSON.stringify(["food"])
    )
  })

  it("does not discard a draft when removing another tag", async () => {
    const user = userEvent.setup()
    render(<Harness initialTags={["travel"]} />)

    const input = screen.getByLabelText("Tags")
    await user.type(input, "food")
    await user.click(screen.getByRole("button", { name: "Remove tag travel" }))

    expect(screen.queryByText("travel")).not.toBeInTheDocument()
    expect(input).toHaveValue("food")
    expect(input).toHaveFocus()
    expect(screen.getByTestId("committed-tags")).toHaveTextContent("[]")

    await user.type(input, ",")

    expect(screen.getByText("food")).toBeInTheDocument()
    expect(screen.getByTestId("committed-tags")).toHaveTextContent(
      JSON.stringify(["food"])
    )
  })
})
