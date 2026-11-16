import { GlobeIcon } from "lucide-react"

import { Badge } from "@/components/ui/badge"

export function PublicShareBadge() {
  return (
    <Badge
      variant="outline"
      className="gap-1 text-[10px] tracking-[0.12em] uppercase"
    >
      <GlobeIcon className="size-3" aria-hidden="true" />
      Public link
    </Badge>
  )
}
