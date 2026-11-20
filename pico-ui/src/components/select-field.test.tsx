import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"

import { SelectField } from "@/components/select-field"

const options = [
  { value: "USD", label: "USD" },
  { value: "INR", label: "INR" },
]

describe("SelectField", () => {
  it("shows the selected label and reports changes", async () => {
    const user = userEvent.setup()
    const onValueChange = vi.fn()

    render(
      <div>
        <label htmlFor="currency">Currency</label>
        <SelectField
          id="currency"
          value="USD"
          onValueChange={onValueChange}
          options={options}
        />
      </div>
    )

    const trigger = screen.getByRole("combobox", { name: "Currency" })
    expect(trigger).toHaveTextContent("USD")

    await user.click(trigger)
    await user.click(screen.getByRole("option", { name: "INR" }))

    expect(onValueChange).toHaveBeenCalledWith("INR")
  })

  it("marks the trigger invalid when requested", () => {
    render(
      <SelectField
        aria-label="Project"
        value=""
        onValueChange={() => {}}
        options={[{ value: "", label: "Select a project" }]}
        invalid
      />
    )

    expect(screen.getByRole("combobox", { name: "Project" })).toHaveAttribute(
      "aria-invalid",
      "true"
    )
  })
})
