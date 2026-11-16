import { GlobeIcon } from "lucide-react"

export function PublicShareBadge() {
  return (
    <span className="inline-flex items-center gap-1 rounded-full border border-border bg-card px-2 py-0.5 text-[10px] font-semibold tracking-[0.12em] text-muted-foreground uppercase">
      <GlobeIcon className="size-3" aria-hidden="true" />
      Public
    </span>
  )
}
