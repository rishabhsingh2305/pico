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
import { NativeSelect, NativeSelectOption } from "@/components/ui/native-select"
import { Spinner } from "@/components/ui/spinner"
import { isUserNotFoundError } from "@/features/auth/session"
import { createProject } from "@/features/projects/api"
import { projectErrorMessage } from "@/features/projects/error-message"
import type { Project } from "@/features/projects/types"
import {
  validateProjectName,
  validateProjectTags,
} from "@/features/projects/validation"
import { CURRENCIES } from "@/lib/currencies"
import { normalizeTags } from "@/lib/tags"

type CreateProjectFormProps = {
  userId: string
  orgId: string
  onCreated: (project: Project) => void
  onCancel: () => void
  onSessionInvalid: () => void
}

export function CreateProjectForm({
  userId,
  orgId,
  onCreated,
  onCancel,
  onSessionInvalid,
}: CreateProjectFormProps) {
  const id = useId()
  const nameId = `${id}-name`
  const nameErrorId = `${id}-name-error`
  const currencyId = `${id}-currency`
  const tagsId = `${id}-tags`
  const tagsHelpId = `${id}-tags-help`
  const tagsErrorId = `${id}-tags-error`

  const [name, setName] = useState("")
  const [currency, setCurrency] = useState<string>("USD")
  const [tags, setTags] = useState<string[]>([])
  const [nameError, setNameError] = useState("")
  const [tagsError, setTagsError] = useState("")
  const [formError, setFormError] = useState("")
  const [isSubmitting, setIsSubmitting] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (isSubmitting) {
      return
    }

    const nextNameError = validateProjectName(name)
    const nextTagsError = validateProjectTags(tags)
    setNameError(nextNameError ?? "")
    setTagsError(nextTagsError ?? "")
    setFormError("")
    if (nextNameError || nextTagsError) {
      return
    }

    setIsSubmitting(true)
    try {
      const project = await createProject(userId, orgId, {
        name: name.trim(),
        currency,
        tags: normalizeTags(tags),
      })
      setName("")
      setTags([])
      setIsSubmitting(false)
      onCreated(project)
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
    <form
      onSubmit={handleSubmit}
      noValidate
      aria-busy={isSubmitting}
      className="flex flex-col gap-4"
    >
      <div className="grid gap-3 sm:grid-cols-[1fr_9rem]">
        <Field data-invalid={Boolean(nameError)}>
          <FieldLabel htmlFor={nameId}>Project name</FieldLabel>
          <Input
            id={nameId}
            name="projectName"
            type="text"
            autoComplete="off"
            placeholder="General"
            value={name}
            onChange={(event) => {
              setName(event.target.value)
              if (nameError) {
                setNameError("")
              }
              if (formError) {
                setFormError("")
              }
            }}
            disabled={isSubmitting}
            aria-invalid={Boolean(nameError)}
            aria-describedby={nameError ? nameErrorId : undefined}
          />
          <FieldError id={nameErrorId}>{nameError}</FieldError>
        </Field>

        <Field>
          <FieldLabel htmlFor={currencyId}>Currency</FieldLabel>
          <NativeSelect
            id={currencyId}
            name="currency"
            className="w-full"
            value={currency}
            onChange={(event) => setCurrency(event.target.value)}
            disabled={isSubmitting}
          >
            {CURRENCIES.map((code) => (
              <NativeSelectOption key={code} value={code}>
                {code}
              </NativeSelectOption>
            ))}
          </NativeSelect>
        </Field>
      </div>

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
          describedBy={tagsError ? tagsErrorId : tagsHelpId}
        />
        <FieldDescription id={tagsHelpId}>
          Type a tag and press comma or Enter. Up to 20 tags.
        </FieldDescription>
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
            "Create project"
          )}
        </Button>
      </div>
    </form>
  )
}
