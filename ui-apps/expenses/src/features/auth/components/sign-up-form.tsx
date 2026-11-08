import { useId, useState, type ChangeEvent, type FormEvent } from "react"
import { CircleAlertIcon, EyeIcon, EyeOffIcon } from "lucide-react"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Spinner } from "@/components/ui/spinner"
import { signUp } from "@/features/auth/api"
import { signUpErrorMessage } from "@/features/auth/error-message"
import type { SignUpInput, User } from "@/features/auth/types"
import {
  validateSignUp,
  type SignUpErrors,
  type SignUpField,
} from "@/features/auth/validation"

type SignUpFormProps = {
  onSuccess: (user: User) => void
}

const emptyForm: SignUpInput = {
  name: "",
  email: "",
  password: "",
}

export function SignUpForm({ onSuccess }: SignUpFormProps) {
  const id = useId()
  const nameId = `${id}-name`
  const emailId = `${id}-email`
  const passwordId = `${id}-password`
  const passwordHintId = `${id}-password-hint`

  const [values, setValues] = useState<SignUpInput>(emptyForm)
  const [errors, setErrors] = useState<SignUpErrors>({})
  const [formError, setFormError] = useState("")
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [showPassword, setShowPassword] = useState(false)

  function updateField(field: SignUpField) {
    return (event: ChangeEvent<HTMLInputElement>) => {
      const value = event.target.value
      setValues((current) => ({ ...current, [field]: value }))
      if (errors[field]) {
        setErrors((current) => ({ ...current, [field]: undefined }))
      }
      if (formError) {
        setFormError("")
      }
    }
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (isSubmitting) {
      return
    }

    const nextErrors = validateSignUp(values)
    setErrors(nextErrors)
    setFormError("")
    if (Object.keys(nextErrors).length > 0) {
      return
    }

    setIsSubmitting(true)
    try {
      const user = await signUp({
        name: values.name.trim(),
        email: values.email.trim().toLowerCase(),
        password: values.password,
      })
      setIsSubmitting(false)
      onSuccess(user)
    } catch (error) {
      setFormError(signUpErrorMessage(error))
      setIsSubmitting(false)
    }
  }

  return (
    <form
      onSubmit={handleSubmit}
      noValidate
      aria-busy={isSubmitting}
      className="flex flex-col gap-5"
    >
      <div className="flex flex-col gap-2">
        <Label htmlFor={nameId}>Name</Label>
        <Input
          id={nameId}
          name="name"
          type="text"
          autoComplete="name"
          autoFocus
          placeholder="Your name"
          className="h-10"
          value={values.name}
          onChange={updateField("name")}
          disabled={isSubmitting}
          aria-invalid={Boolean(errors.name)}
          aria-describedby={errors.name ? `${id}-name-error` : undefined}
        />
        {errors.name ? (
          <p id={`${id}-name-error`} className="text-sm text-destructive">
            {errors.name}
          </p>
        ) : null}
      </div>

      <div className="flex flex-col gap-2">
        <Label htmlFor={emailId}>Email</Label>
        <Input
          id={emailId}
          name="email"
          type="email"
          inputMode="email"
          autoComplete="email"
          placeholder="you@example.com"
          className="h-10"
          value={values.email}
          onChange={updateField("email")}
          disabled={isSubmitting}
          aria-invalid={Boolean(errors.email)}
          aria-describedby={errors.email ? `${id}-email-error` : undefined}
        />
        {errors.email ? (
          <p id={`${id}-email-error`} className="text-sm text-destructive">
            {errors.email}
          </p>
        ) : null}
      </div>

      <div className="flex flex-col gap-2">
        <Label htmlFor={passwordId}>Password</Label>
        <div className="relative">
          <Input
            id={passwordId}
            name="password"
            type={showPassword ? "text" : "password"}
            autoComplete="new-password"
            placeholder="At least 8 characters"
            className="h-10 pr-10"
            value={values.password}
            onChange={updateField("password")}
            disabled={isSubmitting}
            aria-invalid={Boolean(errors.password)}
            aria-describedby={
              errors.password ? `${id}-password-error` : passwordHintId
            }
          />
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            className="absolute top-1/2 right-1 -translate-y-1/2"
            onClick={() => setShowPassword((current) => !current)}
            disabled={isSubmitting}
            aria-label={showPassword ? "Hide password" : "Show password"}
          >
            {showPassword ? (
              <EyeOffIcon aria-hidden="true" />
            ) : (
              <EyeIcon aria-hidden="true" />
            )}
          </Button>
        </div>
        <p id={passwordHintId} className="text-sm text-muted-foreground">
          Use at least 8 characters.
        </p>
        {errors.password ? (
          <p id={`${id}-password-error`} className="text-sm text-destructive">
            {errors.password}
          </p>
        ) : null}
      </div>

      {formError ? (
        <Alert variant="destructive">
          <CircleAlertIcon aria-hidden="true" />
          <AlertDescription>{formError}</AlertDescription>
        </Alert>
      ) : null}

      <Button
        type="submit"
        size="lg"
        className="h-11 w-full"
        disabled={isSubmitting}
      >
        {isSubmitting ? (
          <>
            <Spinner data-icon="inline-start" />
            Creating account...
          </>
        ) : (
          "Create account"
        )}
      </Button>
    </form>
  )
}
