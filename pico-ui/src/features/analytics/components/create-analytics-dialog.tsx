import { useId, useState, type FormEvent } from "react"
import { CircleAlertIcon, PlusIcon } from "lucide-react"

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
import {
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { createAnalyticsSource } from "@/features/analytics/api"
import { analyticsErrorMessage } from "@/features/analytics/error-message"
import type { AnalyticsSource } from "@/features/analytics/types"
import {
  normalizeAnalyticsEvents,
  validateAnalyticsForm,
  type AnalyticsFormErrors,
} from "@/features/analytics/validation"
import { isSessionInvalidError } from "@/features/auth/session"

type CreateAnalyticsDialogProps = {
  orgId: string
  onCreated: (source: AnalyticsSource) => void
  onSessionInvalid: () => void
}

export function CreateAnalyticsDialog({
  orgId,
  onCreated,
  onSessionInvalid,
}: CreateAnalyticsDialogProps) {
  const id = useId()
  const nameId = `${id}-name`
  const descriptionId = `${id}-description`
  const eventsId = `${id}-events`
  const eventsHelpId = `${id}-events-help`
  const eventsErrorId = `${id}-events-error`

  const [open, setOpen] = useState(false)
  const [name, setName] = useState("")
  const [description, setDescription] = useState("")
  const [events, setEvents] = useState<string[]>(["view"])
  const [errors, setErrors] = useState<AnalyticsFormErrors>({})
  const [formError, setFormError] = useState("")
  const [isSubmitting, setIsSubmitting] = useState(false)

  function reset() {
    setName("")
    setDescription("")
    setEvents(["view"])
    setErrors({})
    setFormError("")
    setIsSubmitting(false)
  }

  function handleOpenChange(next: boolean) {
    if (!next && isSubmitting) {
      return
    }
    setOpen(next)
    if (!next) {
      reset()
    }
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (isSubmitting) {
      return
    }
    const nextErrors = validateAnalyticsForm({ name, description, events })
    setErrors(nextErrors)
    setFormError("")
    if (Object.keys(nextErrors).length > 0) {
      return
    }
    setIsSubmitting(true)
    try {
      const source = await createAnalyticsSource(orgId, {
        name: name.trim(),
        description: description.trim(),
        events: normalizeAnalyticsEvents(events),
      })
      setOpen(false)
      reset()
      onCreated(source)
    } catch (error) {
      if (isSessionInvalidError(error)) {
        onSessionInvalid()
        return
      }
      setFormError(analyticsErrorMessage(error))
      setIsSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger
        render={
          <Button
            type="button"
            className="h-10 px-4 text-xs font-semibold tracking-[0.14em] uppercase"
          />
        }
      >
        <PlusIcon data-icon="inline-start" />
        New analytics
      </DialogTrigger>
      <DialogContent className="max-h-[calc(100svh-2rem)] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle className="font-display text-2xl tracking-tight">
            Create analytics
          </DialogTitle>
          <DialogDescription>
            Create a pixel URL, then fire allowlisted events at it.
          </DialogDescription>
        </DialogHeader>
        <form
          onSubmit={handleSubmit}
          noValidate
          aria-busy={isSubmitting}
          className="flex flex-col gap-4"
        >
          <Field data-invalid={Boolean(errors.name)}>
            <FieldLabel htmlFor={nameId}>Name</FieldLabel>
            <Input
              id={nameId}
              name="name"
              value={name}
              onChange={(event) => {
                setName(event.target.value)
                setErrors((current) => ({ ...current, name: undefined }))
              }}
              placeholder="Homepage CTA"
              disabled={isSubmitting}
              aria-invalid={Boolean(errors.name)}
              aria-describedby={errors.name ? `${id}-name-error` : undefined}
            />
            <FieldError id={`${id}-name-error`}>{errors.name}</FieldError>
          </Field>

          <Field data-invalid={Boolean(errors.description)}>
            <FieldLabel htmlFor={descriptionId}>Description</FieldLabel>
            <Textarea
              id={descriptionId}
              name="description"
              value={description}
              onChange={(event) => {
                setDescription(event.target.value)
                setErrors((current) => ({
                  ...current,
                  description: undefined,
                }))
              }}
              placeholder="Where this pixel is used"
              disabled={isSubmitting}
              rows={3}
              aria-invalid={Boolean(errors.description)}
              aria-describedby={
                errors.description ? `${id}-description-error` : undefined
              }
            />
            <FieldError id={`${id}-description-error`}>
              {errors.description}
            </FieldError>
          </Field>

          <Field data-invalid={Boolean(errors.events)}>
            <FieldLabel htmlFor={eventsId}>Allowed events</FieldLabel>
            <TagInput
              id={eventsId}
              name="events"
              value={events}
              onChange={(next) => {
                setEvents(next)
                setErrors((current) => ({ ...current, events: undefined }))
              }}
              placeholder="view, click, signup"
            />
            <FieldDescription id={eventsHelpId}>
              Lowercase letters, numbers, dots, underscores, and hyphens.
            </FieldDescription>
            <FieldError id={eventsErrorId}>{errors.events}</FieldError>
          </Field>

          {formError ? (
            <Alert variant="destructive">
              <CircleAlertIcon aria-hidden="true" />
              <AlertDescription>{formError}</AlertDescription>
            </Alert>
          ) : null}

          <div className="flex justify-end gap-2">
            <Button
              type="button"
              variant="ghost"
              onClick={() => handleOpenChange(false)}
              disabled={isSubmitting}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? "Creating..." : "Create analytics"}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
