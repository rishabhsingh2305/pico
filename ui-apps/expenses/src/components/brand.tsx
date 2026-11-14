import { cn } from "@/lib/utils"

type BrandProps = {
  className?: string
  tone?: "default" | "inverse"
}

export function Brand({ className, tone = "default" }: BrandProps) {
  return (
    <div className={cn("flex items-center gap-2.5", className)}>
      <span
        className={cn(
          "flex size-9 items-center justify-center rounded-xl",
          tone === "inverse"
            ? "bg-primary-foreground text-primary"
            : "bg-primary text-primary-foreground"
        )}
      >
        <span className="font-display text-lg leading-none font-bold">E</span>
      </span>
      <span
        className={cn(
          "font-display text-xl font-bold tracking-tight",
          tone === "inverse" ? "text-primary-foreground" : "text-foreground"
        )}
      >
        Expenses
      </span>
    </div>
  )
}
