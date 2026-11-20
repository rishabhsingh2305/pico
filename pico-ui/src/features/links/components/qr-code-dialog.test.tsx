import { forwardRef } from "react"
import { render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { QRCodeDialog } from "@/features/links/components/qr-code-dialog"
import type { ShortLink } from "@/features/links/types"

vi.mock("qrcode.react", () => ({
  QRCodeCanvas: forwardRef<
    HTMLCanvasElement,
    { value: string; title?: string }
  >(({ value, title }, ref) => (
    <canvas
      ref={ref}
      data-testid="qr-canvas"
      data-value={value}
      aria-label={title}
    />
  )),
}))

const link: ShortLink = {
  id: "link-1",
  orgId: "org-1",
  userId: "user-1",
  slug: "launch",
  url: "https://example.com/page",
  title: "Launch page",
  tags: ["marketing"],
  shortPath: "/s/launch",
  shortUrl: "https://api.example.com/s/launch",
  disabled: false,
  createdAt: "2026-11-20T10:00:00Z",
  updatedAt: "2026-11-20T10:00:00Z",
}

describe("QRCodeDialog", () => {
  it("encodes the short URL and shows both URLs", () => {
    render(<QRCodeDialog link={link} onClose={() => undefined} />)

    expect(screen.getByRole("heading", { name: "QR code" })).toBeInTheDocument()
    expect(screen.getByTestId("qr-canvas")).toHaveAttribute(
      "data-value",
      link.shortUrl
    )
    expect(screen.getByText(link.shortUrl)).toBeInTheDocument()
    expect(screen.getByText(link.url)).toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: "Download PNG" })
    ).toBeInTheDocument()
  })
})
