import { useId, useState, type FormEvent } from "react"
import { CircleAlertIcon } from "lucide-react"

import { CopyButton } from "@/components/copy-button"
import { SelectField } from "@/components/select-field"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Spinner } from "@/components/ui/spinner"
import { isSessionInvalidError } from "@/features/auth/session"
import { buildShareUrl, createShare } from "@/features/sharing/api"
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

type ProjectOption = {
  id: string
  name: string
}

type CreateShareDialogProps = {
  orgId: string
  projects: ProjectOption[]
  initialResourceType?: "org" | "project"
  initialProjectID?: string
  onCreated: (share: ShareCredential) => void
  onCancel: () => void
  onSessionInvalid: () => void
}

export function CreateShareDialog({
  orgId,
  projects,
  initialResourceType = "org",
  initialProjectID = "",
  onCreated,
  onCancel,
  onSessionInvalid,
}: CreateShareDialogProps) {
  const id = useId()
  const resourceId = `${id}-resource`
  const projectId = `${id}-project`
  const nameId = `${id}-name`
  const expiryId = `${id}-expiry`

  const [resourceType, setResourceType] =
    useState<Exclude<ShareResourceType, "expense">>(initialResourceType)
  const [selectedProjectID, setSelectedProjectID] = useState(initialProjectID)
  const [name, setName] = useState("")
  const [expiry, setExpiry] = useState("30")
  const [createdUrl, setCreatedUrl] = useState("")
  const [error, setError] = useState("")
  const [projectError, setProjectError] = useState("")
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (submitting || createdUrl) {
      return
    }
    if (resourceType === "project" && selectedProjectID === "") {
      setProjectError("Select a project.")
      return
    }

    setSubmitting(true)
    setError("")
    setProjectError("")
    try {
      const days = Number(expiry)
      const expiresAt = new Date(
        Date.now() + days * 24 * 60 * 60 * 1000
      ).toISOString()
      const share = await createShare(orgId, {
        resourceType,
        resourceId: resourceType === "org" ? undefined : selectedProjectID,
        name: name.trim(),
        expiresAt,
      })
      onCreated(share)
      if (!share.token) {
        setError("The server did not return a share link.")
        return
      }
      setCreatedUrl(
        buildShareUrl(
          share.token,
          orgId,
          share.resourceType,
          share.resourceId,
          share.projectId
        )
      )
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

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !submitting) {
          onCancel()
        }
      }}
    >
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle className="font-display text-2xl tracking-tight">
            Create share link
          </DialogTitle>
          <DialogDescription>
            Share a read-only view of this organization or one of its projects.
            The link is shown once.
          </DialogDescription>
        </DialogHeader>

        {createdUrl ? (
          <div className="flex flex-col gap-3">
            <Field>
              <FieldLabel htmlFor={`${id}-created`}>Share link</FieldLabel>
              <div className="flex gap-2">
                <Input
                  id={`${id}-created`}
                  readOnly
                  value={createdUrl}
                  className="font-mono text-xs"
                />
                <CopyButton value={createdUrl} label="Copy share link" />
              </div>
              <FieldDescription>
                Copy this link before closing. The server will not return it
                again.
              </FieldDescription>
            </Field>
            <div className="flex justify-end">
              <Button type="button" onClick={onCancel}>
                Done
              </Button>
            </div>
          </div>
        ) : (
          <form
            onSubmit={handleSubmit}
            noValidate
            aria-busy={submitting}
            className="flex flex-col gap-4"
          >
            <div className="grid gap-3 sm:grid-cols-2">
              <Field>
                <FieldLabel htmlFor={resourceId}>Resource</FieldLabel>
                <SelectField
                  id={resourceId}
                  value={resourceType}
                  onValueChange={(value) => {
                    if (value === "org" || value === "project") {
                      setResourceType(value)
                      setProjectError("")
                      setError("")
                    }
                  }}
                  options={[
                    { value: "org", label: "Organization" },
                    { value: "project", label: "Project" },
                  ]}
                  disabled={submitting}
                />
              </Field>

              {resourceType === "project" ? (
                <Field data-invalid={Boolean(projectError)}>
                  <FieldLabel htmlFor={projectId}>Project</FieldLabel>
                  <SelectField
                    id={projectId}
                    value={selectedProjectID}
                    onValueChange={(value) => {
                      setSelectedProjectID(value)
                      setProjectError("")
                      setError("")
                    }}
                    options={[
                      { value: "", label: "Select a project" },
                      ...projects.map((project) => ({
                        value: project.id,
                        label: project.name,
                      })),
                    ]}
                    disabled={submitting}
                    invalid={Boolean(projectError)}
                  />
                  <FieldError>{projectError}</FieldError>
                </Field>
              ) : null}
            </div>

            <Field>
              <FieldLabel htmlFor={nameId}>Name</FieldLabel>
              <Input
                id={nameId}
                value={name}
                onChange={(event) => {
                  setName(event.target.value)
                  setError("")
                }}
                placeholder="Client review"
                disabled={submitting}
              />
              <FieldDescription>Optional label for this link.</FieldDescription>
            </Field>

            <Field>
              <FieldLabel htmlFor={expiryId}>Expiry</FieldLabel>
              <SelectField
                id={expiryId}
                value={expiry}
                onValueChange={setExpiry}
                options={EXPIRY_OPTIONS}
                disabled={submitting}
              />
            </Field>

            {error ? (
              <Alert variant="destructive">
                <CircleAlertIcon aria-hidden="true" />
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            ) : null}

            <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
              <Button
                type="button"
                variant="outline"
                onClick={onCancel}
                disabled={submitting}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={submitting}>
                {submitting ? <Spinner data-icon="inline-start" /> : null}
                {submitting ? "Creating..." : "Create link"}
              </Button>
            </div>
          </form>
        )}
      </DialogContent>
    </Dialog>
  )
}
