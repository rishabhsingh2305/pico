import {
  BanIcon,
  CircleCheckIcon,
  ExternalLinkIcon,
  PencilIcon,
  QrCodeIcon,
  Trash2Icon,
} from "lucide-react"

import { CopyButton } from "@/components/copy-button"
import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import type { ShortLink } from "@/features/links/types"

type LinkListProps = {
  links: ShortLink[]
  clicksByLinkId: ReadonlyMap<string, number>
  statsLoading: boolean
  statsError: boolean
  canWrite: boolean
  canDelete: boolean
  busyId: string | null
  onEdit: (link: ShortLink) => void
  onShowQR: (link: ShortLink) => void
  onToggleDisabled: (link: ShortLink) => void
  onDelete: (link: ShortLink) => void
}

export function LinkList({
  links,
  clicksByLinkId,
  statsLoading,
  statsError,
  canWrite,
  canDelete,
  busyId,
  onEdit,
  onShowQR,
  onToggleDisabled,
  onDelete,
}: LinkListProps) {
  if (links.length === 0) {
    return null
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Link</TableHead>
          <TableHead>Destination</TableHead>
          <TableHead>Tags</TableHead>
          <TableHead>Status</TableHead>
          <TableHead className="text-right">Clicks (30d)</TableHead>
          <TableHead>Created</TableHead>
          <TableHead className="text-right">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {links.map((link) => (
          <TableRow key={link.id}>
            <TableCell className="max-w-[17rem]">
              <LinkTitle link={link} />
            </TableCell>
            <TableCell className="max-w-[18rem]">
              <Destination link={link} />
            </TableCell>
            <TableCell>
              <LinkTags tags={link.tags} />
            </TableCell>
            <TableCell>
              <LinkStatus disabled={link.disabled} />
            </TableCell>
            <TableCell className="text-right text-muted-foreground tabular-nums">
              <ClickCount
                count={clicksByLinkId.get(link.id)}
                loading={statsLoading}
                error={statsError}
              />
            </TableCell>
            <TableCell className="text-muted-foreground">
              {formatCreatedAt(link.createdAt)}
            </TableCell>
            <TableCell>
              <LinkActions
                link={link}
                canWrite={canWrite}
                canDelete={canDelete}
                busy={busyId === link.id}
                onEdit={onEdit}
                onShowQR={onShowQR}
                onToggleDisabled={onToggleDisabled}
                onDelete={onDelete}
              />
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

function LinkTitle({ link }: { link: ShortLink }) {
  return (
    <div className="flex min-w-0 flex-col gap-0.5">
      <a
        href={link.shortUrl}
        target="_blank"
        rel="noreferrer"
        className="truncate font-medium text-foreground underline-offset-4 hover:underline focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
      >
        {displayTitle(link)}
      </a>
      <span className="truncate font-mono text-xs text-muted-foreground">
        {link.shortPath}
      </span>
    </div>
  )
}

function Destination({ link }: { link: ShortLink }) {
  return (
    <span
      title={link.url}
      className="block max-w-[18rem] truncate text-sm text-muted-foreground"
    >
      {link.url}
    </span>
  )
}

function LinkTags({ tags }: { tags: string[] }) {
  if (tags.length === 0) {
    return <span className="text-xs text-muted-foreground">—</span>
  }

  return (
    <div className="flex flex-wrap gap-1">
      {tags.slice(0, 3).map((tag) => (
        <Badge key={tag} variant="outline" className="max-w-24 truncate">
          {tag}
        </Badge>
      ))}
      {tags.length > 3 ? (
        <Badge variant="secondary">+{tags.length - 3}</Badge>
      ) : null}
    </div>
  )
}

function LinkStatus({ disabled }: { disabled: boolean }) {
  return (
    <Badge variant={disabled ? "secondary" : "default"}>
      {disabled ? "Disabled" : "Active"}
    </Badge>
  )
}

function ClickCount({
  count,
  loading,
  error,
}: {
  count: number | undefined
  loading: boolean
  error: boolean
}) {
  if (loading) {
    return <span aria-label="Loading clicks">—</span>
  }
  if (error) {
    return (
      <span title="Could not load click counts" aria-label="Clicks unavailable">
        —
      </span>
    )
  }
  return <>{formatClickCount(count ?? 0)}</>
}

type LinkActionsProps = {
  link: ShortLink
  canWrite: boolean
  canDelete: boolean
  busy: boolean
  onEdit: (link: ShortLink) => void
  onShowQR: (link: ShortLink) => void
  onToggleDisabled: (link: ShortLink) => void
  onDelete: (link: ShortLink) => void
}

function LinkActions({
  link,
  canWrite,
  canDelete,
  busy,
  onEdit,
  onShowQR,
  onToggleDisabled,
  onDelete,
}: LinkActionsProps) {
  return (
    <div className="flex items-center justify-end gap-0.5">
      <CopyButton
        value={link.shortUrl}
        label={`Copy short URL for ${link.slug}`}
      />
      <a
        href={link.shortUrl}
        target="_blank"
        rel="noreferrer"
        aria-label={`Open ${link.slug}`}
        title="Open"
        className={buttonVariants({ variant: "ghost", size: "icon-sm" })}
      >
        <ExternalLinkIcon aria-hidden="true" />
      </a>
      <Button
        type="button"
        variant="ghost"
        size="icon-sm"
        aria-label={`Show QR code for ${link.slug}`}
        title="QR code"
        onClick={() => onShowQR(link)}
      >
        <QrCodeIcon aria-hidden="true" />
      </Button>
      {canWrite ? (
        <>
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            aria-label={`Edit ${link.slug}`}
            title="Edit"
            onClick={() => onEdit(link)}
          >
            <PencilIcon aria-hidden="true" />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            aria-label={`${link.disabled ? "Enable" : "Disable"} ${link.slug}`}
            title={link.disabled ? "Enable" : "Disable"}
            disabled={busy}
            onClick={() => onToggleDisabled(link)}
          >
            {link.disabled ? (
              <CircleCheckIcon aria-hidden="true" />
            ) : (
              <BanIcon aria-hidden="true" />
            )}
          </Button>
        </>
      ) : null}
      {canDelete ? (
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          aria-label={`Delete ${link.slug}`}
          title="Delete"
          disabled={busy}
          className="text-destructive hover:bg-destructive/10 hover:text-destructive"
          onClick={() => onDelete(link)}
        >
          <Trash2Icon aria-hidden="true" />
        </Button>
      ) : null}
    </div>
  )
}

function displayTitle(link: ShortLink): string {
  if (link.title.trim() !== "") {
    return link.title
  }
  try {
    return new URL(link.url).hostname
  } catch {
    return link.slug
  }
}

function formatClickCount(value: number): string {
  return new Intl.NumberFormat().format(value)
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
