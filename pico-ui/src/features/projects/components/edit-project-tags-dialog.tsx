import { useId, useState, type FormEvent } from "react"
import { CircleAlertIcon, TagsIcon } from "lucide-react"

import { TagInput } from "@/components/tag-input"
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
import { Field, FieldError, FieldLabel } from "@/components/ui/field"
import { Spinner } from "@/components/ui/spinner"
import { isUserNotFoundError } from "@/features/auth/session"
import { updateProjectTags } from "@/features/projects/api"
import { projectErrorMessage } from "@/features/projects/error-message"
import type { Project } from "@/features/projects/types"
import { validateProjectTags } from "@/features/projects/validation"
import { normalizeTags } from "@/lib/tags"

type EditProjectTagsDialogProps = {
  userId: string
  orgId: string
  project: Project
  onUpdated: () => void
  onSessionInvalid: () => void
}

export function EditProjectTagsDialog({
  userId,
  orgId,
  project,
  onUpdated,
  onSessionInvalid,
}: EditProjectTagsDialogProps) {
  const triggerId = useId()
  const tagsId = useId()
  const tagsErrorId = `${tagsId}-error`

  const [open, setOpen] = useState(false)
  const [tags, setTags] = useState<string[]>(project.tags)
  const [tagsError, setTagsError] = useState("")
  const [formError, setFormError] = useState("")
  const [isSubmitting, setIsSubmitting] = useState(false)

  function handleOpenChange(nextOpen: boolean) {
    if (isSubmitting) {
      return
    }
    setOpen(nextOpen)
    if (nextOpen) {
      setTags(project.tags)
      setTagsError("")
      setFormError("")
    }
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (isSubmitting) {
      return
    }

    const nextTagsError = validateProjectTags(tags)
    setTagsError(nextTagsError ?? "")
    setFormError("")
    if (nextTagsError) {
      return
    }

    setIsSubmitting(true)
    try {
      await updateProjectTags(userId, orgId, project.id, normalizeTags(tags))
      setIsSubmitting(false)
      setOpen(false)
      onUpdated()
    } catch (error) {
      if (isUserNotFoundError(error)) {
        onSessionInvalid()
        return
      }
      setFormError(projectErrorMessage(error))
      setIsSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange} triggerId={triggerId}>
      <DialogTrigger
        id={triggerId}
        render={
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            aria-label={`Edit tags for ${project.name}`}
          />
        }
      >
        <TagsIcon aria-hidden="true" />
      </DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Edit project tags</DialogTitle>
          <DialogDescription>
            Choose tags for {project.name}. A project matches when any selected
            filter tag matches.
          </DialogDescription>
        </DialogHeader>

        <form
          onSubmit={handleSubmit}
          noValidate
          aria-busy={isSubmitting}
          className="flex flex-col gap-4"
        >
          <Field data-invalid={Boolean(tagsError)}>
            <FieldLabel htmlFor={tagsId}>Tags</FieldLabel>
            <TagInput
              id={tagsId}
              name="tags"
              value={tags}
              onChange={(nextTags) => {
                setTags(nextTags)
                setTagsError("")
                setFormError("")
              }}
              placeholder="travel, food"
              disabled={isSubmitting}
              invalid={Boolean(tagsError)}
              describedBy={tagsError ? tagsErrorId : undefined}
            />
            <FieldError id={tagsErrorId}>{tagsError}</FieldError>
          </Field>

          {formError ? (
            <Alert variant="destructive">
              <CircleAlertIcon aria-hidden="true" />
              <AlertDescription>{formError}</AlertDescription>
            </Alert>
          ) : null}

          <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
            <Button
              type="button"
              variant="outline"
              onClick={() => setOpen(false)}
              disabled={isSubmitting}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? (
                <>
                  <Spinner data-icon="inline-start" />
                  Saving...
                </>
              ) : (
                "Save tags"
              )}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
