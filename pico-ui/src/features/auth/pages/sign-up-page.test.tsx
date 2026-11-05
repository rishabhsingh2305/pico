import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter } from "react-router"
import { afterEach, describe, expect, it, vi } from "vitest"

import { SignUpPage } from "@/features/auth/pages/sign-up-page"

const user = {
  id: "user-1",
  name: "Ada Lovelace",
  email: "ada@example.com",
  emailVerified: false,
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
}

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  } as Response
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/signup"]}>
      <SignUpPage />
    </MemoryRouter>
  )
}

async function fillValidForm() {
  const userEventInstance = userEvent.setup()
  await userEventInstance.type(screen.getByLabelText("Name"), "Ada Lovelace")
  await userEventInstance.type(
    screen.getByLabelText("Email"),
    "ada@example.com"
  )
  await userEventInstance.type(
    screen.getByLabelText("Password"),
    "correct horse"
  )
  return userEventInstance
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe("SignUpPage", () => {
  it("validates required fields before calling the API", async () => {
    const fetchMock = vi.fn<typeof fetch>()
    vi.stubGlobal("fetch", fetchMock)
    const userEventInstance = userEvent.setup()

    renderPage()
    expect(
      screen.getByRole("link", { name: "API documentation" })
    ).toHaveAttribute("href", "/api-docs")
    await userEventInstance.click(
      screen.getByRole("button", { name: "Create account" })
    )

    expect(screen.getByText("Enter your name.")).toBeInTheDocument()
    expect(screen.getByText("Enter your email address.")).toBeInTheDocument()
    expect(
      screen.getByText("Password must be at least 8 characters.")
    ).toBeInTheDocument()
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it("submits the form and shows the success state", async () => {
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(jsonResponse(user, 201))
    vi.stubGlobal("fetch", fetchMock)

    renderPage()
    const userEventInstance = await fillValidForm()
    await userEventInstance.click(
      screen.getByRole("button", { name: "Create account" })
    )

    expect(
      await screen.findByRole("heading", { name: "Check your email" })
    ).toBeInTheDocument()
    expect(screen.getByText("ada@example.com")).toBeInTheDocument()

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe("/api/users")
    expect(init?.method).toBe("POST")
    expect(new Headers(init?.headers).get("Content-Type")).toBe(
      "application/json"
    )
    expect(JSON.parse(String(init?.body))).toEqual({
      name: "Ada Lovelace",
      email: "ada@example.com",
      password: "correct horse",
    })
  })

  it("allows the user to start over after signing up", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn<typeof fetch>().mockResolvedValueOnce(jsonResponse(user, 201))
    )

    renderPage()
    const userEventInstance = await fillValidForm()
    await userEventInstance.click(
      screen.getByRole("button", { name: "Create account" })
    )
    await screen.findByRole("heading", { name: "Check your email" })

    await userEventInstance.click(
      screen.getByRole("button", { name: "Use a different email" })
    )

    expect(
      screen.getByRole("heading", { name: "Create your account" })
    ).toBeInTheDocument()
    expect(screen.getByLabelText("Name")).toHaveValue("")
  })

  it("shows duplicate email errors", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn<typeof fetch>()
        .mockResolvedValueOnce(jsonResponse({ error: "email exists" }, 409))
    )

    renderPage()
    const userEventInstance = await fillValidForm()
    await userEventInstance.click(
      screen.getByRole("button", { name: "Create account" })
    )

    expect(
      await screen.findByText("An account with this email already exists.")
    ).toBeInTheDocument()
  })

  it("shows email delivery errors", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn<typeof fetch>()
        .mockResolvedValueOnce(jsonResponse({ error: "delivery failed" }, 502))
    )

    renderPage()
    const userEventInstance = await fillValidForm()
    await userEventInstance.click(
      screen.getByRole("button", { name: "Create account" })
    )

    expect(
      await screen.findByText(
        "We couldn't send the verification email. Please try again."
      )
    ).toBeInTheDocument()
  })

  it("shows server validation messages", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn<typeof fetch>()
        .mockResolvedValueOnce(jsonResponse({ error: "email is invalid" }, 400))
    )

    renderPage()
    const userEventInstance = await fillValidForm()
    await userEventInstance.click(
      screen.getByRole("button", { name: "Create account" })
    )

    expect(await screen.findByText("email is invalid")).toBeInTheDocument()
  })

  it("shows network errors", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn<typeof fetch>()
        .mockRejectedValueOnce(new TypeError("Failed to fetch"))
    )

    renderPage()
    const userEventInstance = await fillValidForm()
    await userEventInstance.click(
      screen.getByRole("button", { name: "Create account" })
    )

    expect(
      await screen.findByText(
        "Could not reach the server. Check your connection and try again."
      )
    ).toBeInTheDocument()
  })

  it("disables the form while the request is in flight", async () => {
    let resolveRequest: (response: Response) => void = () => {}
    vi.stubGlobal(
      "fetch",
      vi.fn<typeof fetch>().mockReturnValueOnce(
        new Promise<Response>((resolve) => {
          resolveRequest = resolve
        })
      )
    )

    renderPage()
    const userEventInstance = await fillValidForm()
    await userEventInstance.click(
      screen.getByRole("button", { name: "Create account" })
    )

    const submit = await screen.findByRole("button", {
      name: /creating account/i,
    })
    expect(submit).toBeDisabled()
    expect(screen.getByLabelText("Name")).toBeDisabled()

    resolveRequest(jsonResponse(user, 201))
    await waitFor(() => {
      expect(
        screen.getByRole("heading", { name: "Check your email" })
      ).toBeInTheDocument()
    })
  })

  it("toggles password visibility", async () => {
    const userEventInstance = userEvent.setup()
    renderPage()

    const password = screen.getByLabelText("Password")
    expect(password).toHaveAttribute("type", "password")

    await userEventInstance.click(
      screen.getByRole("button", { name: "Show password" })
    )

    expect(password).toHaveAttribute("type", "text")
    expect(
      screen.getByRole("button", { name: "Hide password" })
    ).toBeInTheDocument()
  })
})
