import { useEffect, useId, useState } from "react"
import { Trash2Icon } from "lucide-react"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import { NativeSelect, NativeSelectOption } from "@/components/ui/native-select"
import { ApiError } from "@/lib/api-client"
import { Spinner } from "@/components/ui/spinner"
import {
  buildShareUrl,
  createShare,
  listShares,
  revokeShare,
} from "@/features/sharing/api"
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
  open: boolean
  onOpenChange: (open: boolean) => void
  orgId: string
  resourceType: ShareResourceType
  resourceId?: string
  resourceName: string
  onChanged?: () => void
  onSessionInvalid: () => void
}

export function ShareDialog({
  open,
  onOpenChange,
  orgId,
  resourceType,
  resourceId,
  resourceName,
  onChanged,
  onSessionInvalid,
}: ShareDialogProps) {
  const selectId = useId()
  const [shares, setShares] = useState<ShareCredential[]>([])
  const [expiry, setExpiry] = useState("30")
  const [createdUrl, setCreatedUrl] = useState("")
  const [createdShareID, setCreatedShareID] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(open)
  const [submitting, setSubmitting] = useState(false)
  const [revokingID, setRevokingID] = useState("")
  const effectiveResourceID = resourceId ?? orgId

  function handleOpenChange(next: boolean) {
    onOpenChange(next)
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
          setError("")
        }
      })
      .catch((caught: unknown) => {
        if (cancelled) {
          return
        }
        if (caught instanceof ApiError && caught.status === 401) {
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
      if (caught instanceof ApiError && caught.status === 401) {
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
      if (caught instanceof ApiError && caught.status === 401) {
        onSessionInvalid()
        return
      }
      setError(shareErrorMessage(caught))
    } finally {
      setRevokingID("")
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Share {resourceName}</DialogTitle>
          <DialogDescription>
            Anyone with the link can view this without signing in. Links are
            read-only and can be revoked at any time.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor={selectId}>Link expiry</Label>
            <NativeSelect
              id={selectId}
              className="w-full"
              value={expiry}
              onChange={(event) => setExpiry(event.target.value)}
            >
              {EXPIRY_OPTIONS.map((option) => (
                <NativeSelectOption key={option.value} value={option.value}>
                  {option.label}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          </div>

          <Button type="button" onClick={handleCreate} disabled={submitting}>
            {submitting ? <Spinner /> : "Create public link"}
          </Button>

          {createdUrl ? (
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="share-link">New link</Label>
              <input
                id="share-link"
                readOnly
                value={createdUrl}
                className="h-9 rounded-md border border-input bg-muted px-3 text-base md:text-sm"
              />
              <Button
                type="button"
                variant="outline"
                onClick={() => void navigator.clipboard.writeText(createdUrl)}
              >
                Copy
              </Button>
            </div>
          ) : null}

          {error ? (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          ) : null}

          <div className="flex flex-col gap-2">
            <p className="text-xs font-semibold text-muted-foreground uppercase">
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
                      <p className="text-sm font-medium capitalize">
                        {share.status}
                      </p>
                      <p className="truncate text-xs text-muted-foreground">
                        {share.expiresAt
                          ? `Expires ${new Date(share.expiresAt).toLocaleDateString()}`
                          : "Default expiry"}
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

function shareErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message) {
    return error.message
  }
  return "Could not update sharing."
}
