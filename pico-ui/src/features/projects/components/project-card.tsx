import { Link } from "react-router"
import { ChevronRightIcon, FolderIcon } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Card, CardContent } from "@/components/ui/card"
import { EditProjectTagsDialog } from "@/features/projects/components/edit-project-tags-dialog"
import type { Project } from "@/features/projects/types"

type ProjectCardProps = {
  userId: string
  orgId: string
  project: Project
  activeTags: string[]
  onToggleTag: (tag: string) => void
  onTagsUpdated: () => void
  onSessionInvalid: () => void
}

export function ProjectCard({
  userId,
  orgId,
  project,
  activeTags,
  onToggleTag,
  onTagsUpdated,
  onSessionInvalid,
}: ProjectCardProps) {
  return (
    <Card
      size="sm"
      className="h-full transition-colors hover:border-primary/20 hover:bg-accent/30"
    >
      <CardContent className="flex h-full flex-col gap-3">
        <div className="flex items-start justify-between gap-3">
          <Link
            to={`/orgs/${orgId}/projects/${project.id}`}
            aria-label={project.name}
            className="group flex min-w-0 items-center gap-3 rounded-sm focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
          >
            <span className="flex size-10 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary">
              <FolderIcon className="size-4" aria-hidden="true" />
            </span>
            <span className="min-w-0">
              <span className="block truncate font-display text-lg font-semibold tracking-tight group-hover:underline">
                {project.name}
              </span>
              <span className="mt-0.5 block text-xs text-muted-foreground">
                {project.currency} · Project
              </span>
            </span>
          </Link>

          <div className="flex shrink-0 items-center gap-1">
            <EditProjectTagsDialog
              userId={userId}
              orgId={orgId}
              project={project}
              onUpdated={onTagsUpdated}
              onSessionInvalid={onSessionInvalid}
            />
            <ChevronRightIcon
              className="size-4 text-muted-foreground"
              aria-hidden="true"
            />
          </div>
        </div>

        {project.tags.length > 0 ? (
          <div className="mt-auto flex flex-wrap gap-1">
            {project.tags.map((tag) => {
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
