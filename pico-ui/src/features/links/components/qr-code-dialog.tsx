import { useRef } from "react"
import { QRCodeCanvas } from "qrcode.react"

import { CopyButton } from "@/components/copy-button"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import type { ShortLink } from "@/features/links/types"

type QRCodeDialogProps = {
  link: ShortLink
  onClose: () => void
}

export function QRCodeDialog({ link, onClose }: QRCodeDialogProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)

  function download() {
    const canvas = canvasRef.current
    if (!canvas) {
      return
    }

    const anchor = document.createElement("a")
    anchor.href = canvas.toDataURL("image/png")
    anchor.download = `pico-link-${link.slug}.png`
    anchor.click()
  }

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose()
        }
      }}
    >
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle className="font-display text-2xl tracking-tight">
            QR code
          </DialogTitle>
          <DialogDescription>
            Scan to open <span className="font-mono">{link.slug}</span>.
          </DialogDescription>
        </DialogHeader>

        <div
          role="img"
          aria-label={`QR code for ${link.shortUrl}`}
          className="mx-auto w-fit rounded-lg bg-white p-4"
        >
          <QRCodeCanvas
            ref={canvasRef}
            value={link.shortUrl}
            size={224}
            level="M"
            marginSize={2}
            title={`QR code for ${link.shortUrl}`}
          />
        </div>

        <div className="flex flex-col gap-1 text-center">
          <p className="font-mono text-xs break-all text-foreground">
            {link.shortUrl}
          </p>
          <p className="text-xs break-all text-muted-foreground">{link.url}</p>
        </div>

        <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <Button type="button" variant="outline" onClick={onClose}>
            Close
          </Button>
          <CopyButton value={link.shortUrl} label="Copy short URL" />
          <Button type="button" onClick={download}>
            Download PNG
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
