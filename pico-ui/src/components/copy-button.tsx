import { useState } from "react"
import { CheckIcon, CircleAlertIcon, CopyIcon } from "lucide-react"

import { Button } from "@/components/ui/button"

type CopyButtonProps = {
  value: string
  label: string
}

type CopyState = "idle" | "copied" | "error"

export function CopyButton({ value, label }: CopyButtonProps) {
  const [state, setState] = useState<CopyState>("idle")

  async function copy() {
    if (!navigator.clipboard) {
      setState("error")
      return
    }

    try {
      await navigator.clipboard.writeText(value)
      setState("copied")
    } catch {
      setState("error")
    }
  }

  return (
    <Button
      type="button"
      variant="ghost"
      size="xs"
      onClick={copy}
      aria-label={label}
    >
      {state === "copied" ? (
        <CheckIcon data-icon="inline-start" aria-hidden="true" />
      ) : state === "error" ? (
        <CircleAlertIcon data-icon="inline-start" aria-hidden="true" />
      ) : (
        <CopyIcon data-icon="inline-start" aria-hidden="true" />
      )}
      <span aria-live="polite">
        {state === "copied"
          ? "Copied"
          : state === "error"
            ? "Copy failed"
            : "Copy"}
      </span>
    </Button>
  )
}
