import { useId, useRef, useState, type FormEvent } from "react"
import { CircleAlertIcon } from "lucide-react"

import { TagInput } from "@/components/tag-input"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
} from "@/components/ui/card"
import {
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Spinner } from "@/components/ui/spinner"
import { isUserNotFoundError } from "@/features/auth/session"
import { createOrg } from "@/features/orgs/api"
import { orgErrorMessage } from "@/features/orgs/error-message"
import type { Org } from "@/features/orgs/types"
import { validateOrgName, validateOrgTags } from "@/features/orgs/validation"
import { normalizeTags } from "@/lib/tags"

type CreateOrgFormProps = {
  userId: string
  onCreated: (org: Org) => void
  onSessionInvalid: () => void
}

export function CreateOrgForm({
  userId,
  onCreated,
  onSessionInvalid,
}: CreateOrgFormProps) {
  const id = useId()
  const nameId = `${id}-name`
  const nameErrorId = `${id}-name-error`
  const tagsId = `${id}-tags`
  const tagsHelpId = `${id}-tags-help`
  const tagsErrorId = `${id}-tags-error`
  const inputRef = useRef<HTMLInputElement>(null)

  const [name, setName] = useState("")
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

    const nextNameError = validateOrgName(name)
    const nextTagsError = validateOrgTags(tags)
    setNameError(nextNameError ?? "")
    setTagsError(nextTagsError ?? "")
    setFormError("")
    if (nextNameError || nextTagsError) {
      return
    }

    setIsSubmitting(true)
    try {
      const org = await createOrg(userId, {
        name: name.trim(),
        tags: normalizeTags(tags),
      })
      setName("")
      setTags([])
      inputRef.current?.focus()
      setIsSubmitting(false)
      onCreated(org)
    } catch (error) {
      if (isUserNotFoundError(error)) {
        onSessionInvalid()
        return
      }
      setFormError(orgErrorMessage(error))
      setIsSubmitting(false)
    }
  }

  return (
    <Card>
      <CardHeader>
        <h2 className="font-display text-xl font-semibold tracking-tight">
          Create organization
        </h2>
        <CardDescription>
          Organizations group your projects and shared expenses.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form
          onSubmit={handleSubmit}
          noValidate
          aria-busy={isSubmitting}
          className="flex flex-col gap-4"
        >
          <Field data-invalid={Boolean(nameError)}>
            <FieldLabel htmlFor={nameId}>Name</FieldLabel>
            <Input
              ref={inputRef}
              id={nameId}
              name="name"
              type="text"
              autoComplete="organization"
              placeholder="Acme Inc."
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
              aria-describedby={nameError ? nameErrorId : `${id}-name-help`}
            />
            <FieldDescription id={`${id}-name-help`}>
              Use a recognizable team or company name.
            </FieldDescription>
            <FieldError id={nameErrorId}>{nameError}</FieldError>
          </Field>

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
              placeholder="team, finance"
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

          <div className="flex justify-end">
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
                "Create organization"
              )}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  )
}
