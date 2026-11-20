import { useId, useState, type FormEvent } from "react"
import { CircleAlertIcon } from "lucide-react"

import { TagInput } from "@/components/tag-input"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import {
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Spinner } from "@/components/ui/spinner"
import { isSessionInvalidError } from "@/features/auth/session"
import { createLink } from "@/features/links/api"
import { linkErrorMessage } from "@/features/links/error-message"
import type { ShortLink } from "@/features/links/types"
import {
  validateLinkForm,
  type LinkFormErrors,
} from "@/features/links/validation"
import { normalizeTags } from "@/lib/tags"

type CreateLinkFormProps = {
  orgId: string
  onCreated: (link: ShortLink) => void
  onCancel: () => void
  onSessionInvalid: () => void
}

export function CreateLinkForm({
  orgId,
  onCreated,
  onCancel,
  onSessionInvalid,
}: CreateLinkFormProps) {
  const id = useId()
  const urlId = `${id}-url`
  const urlHelpId = `${id}-url-help`
  const urlErrorId = `${id}-url-error`
  const slugId = `${id}-slug`
  const slugHelpId = `${id}-slug-help`
  const slugErrorId = `${id}-slug-error`
  const titleId = `${id}-title`
  const titleErrorId = `${id}-title-error`
  const tagsId = `${id}-tags`
  const tagsHelpId = `${id}-tags-help`
  const tagsErrorId = `${id}-tags-error`

  const [url, setUrl] = useState("")
  const [slug, setSlug] = useState("")
  const [title, setTitle] = useState("")
  const [tags, setTags] = useState<string[]>([])
  const [errors, setErrors] = useState<LinkFormErrors>({})
  const [formError, setFormError] = useState("")
  const [isSubmitting, setIsSubmitting] = useState(false)

  function clearError(field: keyof LinkFormErrors) {
    if (errors[field]) {
      setErrors((current) => ({ ...current, [field]: undefined }))
    }
    if (formError) {
      setFormError("")
    }
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (isSubmitting) {
      return
    }

    const nextErrors = validateLinkForm({ url, slug, title, tags })
    setErrors(nextErrors)
    setFormError("")
    if (Object.keys(nextErrors).length > 0) {
      return
    }

    setIsSubmitting(true)
    try {
      const link = await createLink(orgId, {
        url: url.trim(),
        slug: slug.trim() === "" ? undefined : slug.trim().toLowerCase(),
        title: title.trim(),
        tags: normalizeTags(tags),
      })
      setUrl("")
      setSlug("")
      setTitle("")
      setTags([])
      setIsSubmitting(false)
      onCreated(link)
    } catch (error) {
      if (isSessionInvalidError(error)) {
        onSessionInvalid()
        return
      }
      setFormError(linkErrorMessage(error))
      setIsSubmitting(false)
    }
  }

  return (
    <form
      onSubmit={handleSubmit}
      noValidate
      aria-busy={isSubmitting}
      className="flex flex-col gap-4"
    >
      <Field data-invalid={Boolean(errors.url)}>
        <FieldLabel htmlFor={urlId}>Destination URL</FieldLabel>
        <Input
          id={urlId}
          name="url"
          type="url"
          inputMode="url"
          autoComplete="url"
          placeholder="https://example.com/page"
          value={url}
          onChange={(event) => {
            setUrl(event.target.value)
            clearError("url")
          }}
          disabled={isSubmitting}
          aria-invalid={Boolean(errors.url)}
          aria-describedby={errors.url ? urlErrorId : urlHelpId}
        />
        <FieldDescription id={urlHelpId}>
          Only http and https links are supported.
        </FieldDescription>
        <FieldError id={urlErrorId}>{errors.url}</FieldError>
      </Field>

      <div className="grid gap-3 sm:grid-cols-2">
        <Field data-invalid={Boolean(errors.slug)}>
          <FieldLabel htmlFor={slugId}>Custom alias</FieldLabel>
          <Input
            id={slugId}
            name="slug"
            type="text"
            autoComplete="off"
            placeholder="launch"
            value={slug}
            onChange={(event) => {
              setSlug(event.target.value)
              clearError("slug")
            }}
            disabled={isSubmitting}
            aria-invalid={Boolean(errors.slug)}
            aria-describedby={errors.slug ? slugErrorId : slugHelpId}
          />
          <FieldDescription id={slugHelpId}>
            Optional. Leave blank for a random alias.
          </FieldDescription>
          <FieldError id={slugErrorId}>{errors.slug}</FieldError>
        </Field>

        <Field data-invalid={Boolean(errors.title)}>
          <FieldLabel htmlFor={titleId}>Title</FieldLabel>
          <Input
            id={titleId}
            name="title"
            type="text"
            autoComplete="off"
            placeholder="Launch page"
            value={title}
            onChange={(event) => {
              setTitle(event.target.value)
              clearError("title")
            }}
            disabled={isSubmitting}
            aria-invalid={Boolean(errors.title)}
            aria-describedby={errors.title ? titleErrorId : undefined}
          />
          <FieldError id={titleErrorId}>{errors.title}</FieldError>
        </Field>
      </div>

      <Field data-invalid={Boolean(errors.tags)}>
        <FieldLabel htmlFor={tagsId}>Tags</FieldLabel>
        <TagInput
          id={tagsId}
          name="tags"
          value={tags}
          onChange={(nextTags) => {
            setTags(nextTags)
            clearError("tags")
          }}
          placeholder="marketing, launch"
          disabled={isSubmitting}
          invalid={Boolean(errors.tags)}
          describedBy={errors.tags ? tagsErrorId : tagsHelpId}
        />
        <FieldDescription id={tagsHelpId}>
          Type a tag and press comma or Enter. Up to 20 tags.
        </FieldDescription>
        <FieldError id={tagsErrorId}>{errors.tags}</FieldError>
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
          onClick={onCancel}
          disabled={isSubmitting}
        >
          Cancel
        </Button>
        <Button
          type="submit"
          disabled={isSubmitting}
          className="w-full sm:w-auto"
        >
          {isSubmitting ? (
            <>
              <Spinner data-icon="inline-start" />
              Creating...
            </>
          ) : (
            "Create link"
          )}
        </Button>
      </div>
    </form>
  )
}
