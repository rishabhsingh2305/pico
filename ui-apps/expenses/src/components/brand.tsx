import { cn } from "@/lib/utils"

type BrandProps = {
  className?: string
  tone?: "default" | "inverse"
}

export function Brand({ className, tone = "default" }: BrandProps) {
  return (
    <div className={cn("flex items-center gap-2", className)}>
      <span
        className={cn(
          "flex size-9 items-center justify-center border-2",
          tone === "inverse"
            ? "border-primary-foreground bg-primary-foreground text-primary"
            : "border-foreground bg-primary text-primary-foreground"
        )}
      >
        <span className="font-display text-lg leading-none">E</span>
      </span>
      <span
        className={cn(
          "font-display text-xl tracking-tight",
          tone === "inverse" ? "text-primary-foreground" : "text-foreground"
        )}
      >
        Expenses
      </span>
    </div>
  )
}
