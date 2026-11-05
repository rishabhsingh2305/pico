import { Link } from "react-router"
import { Building2Icon, ChevronRightIcon } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Card, CardContent } from "@/components/ui/card"
import { EditOrgTagsDialog } from "@/features/orgs/components/edit-org-tags-dialog"
import type { Org } from "@/features/orgs/types"

type OrgCardProps = {
  userId: string
  org: Org
  activeTags: string[]
  onToggleTag: (tag: string) => void
  onTagsUpdated: () => void
  onSessionInvalid: () => void
}

export function OrgCard({
  userId,
  org,
  activeTags,
  onToggleTag,
  onTagsUpdated,
  onSessionInvalid,
}: OrgCardProps) {
  const isOwner = org.userId === userId

  return (
    <Card
      size="sm"
      className="h-full transition-colors hover:border-primary/20 hover:bg-accent/30"
    >
      <CardContent className="flex h-full flex-col gap-3">
        <div className="flex items-start justify-between gap-3">
          <Link
            to={`/orgs/${org.id}`}
            aria-label={org.name}
            className="group flex min-w-0 items-center gap-3 rounded-sm focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
          >
            <span className="flex size-10 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary">
              <Building2Icon className="size-4" aria-hidden="true" />
            </span>
            <span className="min-w-0">
              <span className="block truncate font-display text-lg font-semibold tracking-tight group-hover:underline">
                {org.name}
              </span>
              <span className="mt-0.5 block text-xs text-muted-foreground">
                Created {formatCreatedAt(org.createdAt)}
              </span>
            </span>
          </Link>

          <div className="flex shrink-0 items-center gap-1">
            <Badge variant={isOwner ? "default" : "secondary"}>
              {isOwner ? "Owner" : "Member"}
            </Badge>
            {isOwner ? (
              <EditOrgTagsDialog
                userId={userId}
                org={org}
                onUpdated={onTagsUpdated}
                onSessionInvalid={onSessionInvalid}
              />
            ) : null}
            <ChevronRightIcon
              className="size-4 text-muted-foreground"
              aria-hidden="true"
            />
          </div>
        </div>

        {org.tags.length > 0 ? (
          <div className="mt-auto flex flex-wrap gap-1">
            {org.tags.map((tag) => {
              const isActive = activeTags.includes(tag)
              return (
                <button
                  key={tag}
                  type="button"
                  aria-label={`Filter by ${tag}`}
                  aria-pressed={isActive}
                  onClick={() => onToggleTag(tag)}
                  className="rounded-full focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
                >
                  <Badge
                    variant={isActive ? "default" : "outline"}
                    className="rounded-sm text-[10px] tracking-[0.12em] uppercase"
                  >
                    {tag}
                  </Badge>
                </button>
              )
            })}
          </div>
        ) : null}
      </CardContent>
    </Card>
  )
}

function formatCreatedAt(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return "recently"
  }
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
  }).format(date)
}
