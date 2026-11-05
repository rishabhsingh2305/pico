import { WalletIcon } from "lucide-react"

import { cn } from "cn"

type BrandProps = {
  className?: string
  align?: "center" | "start"
  tone?: "default" | "inverse"
}

export function Brand({
  className,
  align = "center",
  tone = "default",
}: BrandProps) {
  return (
    <div
      className={cn(
        "flex items-center gap-2.5",
        align === "center" ? "justify-center" : "justify-start",
        className
      )}
    >
      <div
        className={cn(
          "flex size-9 items-center justify-center rounded-md",
          tone === "inverse"
            ? "bg-primary-foreground text-primary"
            : "bg-primary text-primary-foreground"
        )}
      >
        <WalletIcon className="size-4" aria-hidden="true" />
      </div>
      <span
        className={cn(
          "font-display text-2xl font-bold tracking-[-0.04em]",
          tone === "inverse" && "text-primary-foreground"
        )}
      >
        Pico
      </span>
    </div>
  )
}
