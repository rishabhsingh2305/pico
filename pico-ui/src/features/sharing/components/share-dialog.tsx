import { useEffect, useId, useState } from "react"
import { Link2Icon, Trash2Icon } from "lucide-react"

import { SelectField } from "@/components/select-field"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Field, FieldLabel } from "@/components/ui/field"
import { Spinner } from "@/components/ui/spinner"
import { isSessionInvalidError } from "@/features/auth/session"
import {
  buildShareUrl,
  createShare,
  listShares,
  revokeShare,
} from "@/features/sharing/api"
import { shareErrorMessage } from "@/features/sharing/error-message"
import type {
  ShareCredential,
  ShareResourceType,
} from "@/features/sharing/types"

const EXPIRY_OPTIONS = [
  { value: "7", label: "7 days" },
  { value: "30", label: "30 days" },
  { value: "365", label: "365 days" },
]

type ShareDialogProps = {
  orgId: string
  resourceType: ShareResourceType
  resourceId?: string
  resourceName: string
  onChanged?: () => void
  onSessionInvalid: () => void
}

export function ShareDialog({
  orgId,
  resourceType,
  resourceId,
  resourceName,
  onChanged,
  onSessionInvalid,
}: ShareDialogProps) {
  const selectId = useId()
  const [open, setOpen] = useState(false)
  const [shares, setShares] = useState<ShareCredential[]>([])
  const [expiry, setExpiry] = useState("30")
  const [createdUrl, setCreatedUrl] = useState("")
  const [createdShareID, setCreatedShareID] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [revokingID, setRevokingID] = useState("")

  const effectiveResourceID = resourceId ?? orgId

  function handleOpenChange(next: boolean) {
    setOpen(next)
    if (next) {
      setLoading(true)
      setError("")
    }
  }

  useEffect(() => {
    if (!open) {
      return
    }
    let cancelled = false
    listShares(orgId, resourceType, effectiveResourceID)
      .then((items) => {
        if (!cancelled) {
          setShares(items)
        }
      })
      .catch((caught: unknown) => {
        if (cancelled) {
          return
        }
        if (isSessionInvalidError(caught)) {
          onSessionInvalid()
          return
        }
        setError(shareErrorMessage(caught))
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false)
        }
      })
    return () => {
      cancelled = true
    }
  }, [open, orgId, resourceType, effectiveResourceID, onSessionInvalid])

  async function handleCreate() {
    if (submitting) {
      return
    }
    setSubmitting(true)
    setError("")
    try {
      const days = Number(expiry)
      const expiresAt = new Date(
        Date.now() + days * 24 * 60 * 60 * 1000
      ).toISOString()
      const share = await createShare(orgId, {
        resourceType,
        resourceId: resourceType === "org" ? undefined : effectiveResourceID,
        expiresAt,
      })
      if (share.token) {
        setCreatedUrl(
          buildShareUrl(
            share.token,
            orgId,
            resourceType,
            effectiveResourceID,
            share.projectId
          )
        )
        setCreatedShareID(share.id)
      }
      setShares((current) => [share, ...current])
      onChanged?.()
    } catch (caught) {
      if (isSessionInvalidError(caught)) {
        onSessionInvalid()
        return
      }
      setError(shareErrorMessage(caught))
    } finally {
      setSubmitting(false)
    }
  }

  async function handleRevoke(share: ShareCredential) {
    setRevokingID(share.id)
    setError("")
    try {
      await revokeShare(orgId, share.id)
      setShares((current) =>
        current.map((item) =>
          item.id === share.id ? { ...item, status: "revoked" } : item
        )
      )
      if (share.id === createdShareID) {
        setCreatedUrl("")
        setCreatedShareID("")
      }
      onChanged?.()
    } catch (caught) {
      if (isSessionInvalidError(caught)) {
        onSessionInvalid()
        return
      }
      setError(shareErrorMessage(caught))
    } finally {
      setRevokingID("")
    }
  }

  async function copyLink() {
    if (!createdUrl) {
      return
    }
    try {
      await navigator.clipboard.writeText(createdUrl)
    } catch {
      setError("Could not copy the link. Select it manually.")
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger
        render={
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            aria-label={`Share ${resourceName}`}
          >
            <Link2Icon aria-hidden="true" />
          </Button>
        }
      />
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="font-display text-2xl tracking-tight">
            Share {resourceName}
          </DialogTitle>
          <DialogDescription>
            Anyone with the link can view this without signing in. Links are
            read-only and can be revoked at any time.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-4">
          <Field>
            <FieldLabel htmlFor={selectId}>Link expiry</FieldLabel>
            <SelectField
              id={selectId}
              value={expiry}
              onValueChange={setExpiry}
              options={EXPIRY_OPTIONS}
              className="w-full"
            />
          </Field>

          <Button type="button" onClick={handleCreate} disabled={submitting}>
            {submitting ? <Spinner /> : "Create public link"}
          </Button>

          {createdUrl ? (
            <div className="flex flex-col gap-2">
              <FieldLabel htmlFor="share-link">New link</FieldLabel>
              <div className="flex gap-2">
                <input
                  id="share-link"
                  readOnly
                  value={createdUrl}
                  className="h-9 min-w-0 flex-1 rounded-md border border-input bg-muted px-3 text-base md:text-sm"
                />
                <Button type="button" variant="outline" onClick={copyLink}>
                  Copy
                </Button>
              </div>
              <p className="text-xs text-muted-foreground">
                Copy this link before leaving the page. The server will not
                return it again.
              </p>
            </div>
          ) : null}

          {error ? (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          ) : null}

          <div className="flex flex-col gap-2">
            <p className="text-label text-muted-foreground uppercase">
              Existing links
            </p>
            {loading ? (
              <Spinner />
            ) : shares.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                No links for this resource.
              </p>
            ) : (
              <ul className="divide-y divide-border rounded-md border border-border">
                {shares.map((share) => (
                  <li
                    key={share.id}
                    className="flex items-center justify-between gap-3 px-3 py-2"
                  >
                    <div className="min-w-0">
                      <p className="text-sm font-medium">
                        {share.status === "active"
                          ? "Active"
                          : share.status === "expired"
                            ? "Expired"
                            : "Revoked"}
                      </p>
                      <p className="truncate text-xs text-muted-foreground">
                        {share.expiresAt
                          ? `Expires ${formatDate(share.expiresAt)}`
                          : "Computed default expiry"}
                      </p>
                    </div>
                    {share.status === "active" ? (
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon-sm"
                        aria-label="Revoke link"
                        disabled={revokingID === share.id}
                        onClick={() => handleRevoke(share)}
                      >
                        {revokingID === share.id ? (
                          <Spinner />
                        ) : (
                          <Trash2Icon aria-hidden="true" />
                        )}
                      </Button>
                    ) : null}
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function formatDate(value: string): string {
  return new Date(value).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  })
}
