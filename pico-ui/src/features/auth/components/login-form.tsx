import { useId, useState, type ChangeEvent, type FormEvent } from "react"
import { CircleAlertIcon, EyeIcon, EyeOffIcon } from "lucide-react"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import {
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Spinner } from "@/components/ui/spinner"
import { useAuth } from "@/features/auth/auth-provider"
import { loginErrorMessage } from "@/features/auth/error-message"
import type { LoginInput } from "@/features/auth/types"
import {
  validateLogin,
  type LoginErrors,
  type LoginField,
} from "@/features/auth/validation"

type LoginFormProps = {
  onSuccess: () => void
}

const emptyForm: LoginInput = {
  email: "",
  password: "",
}

export function LoginForm({ onSuccess }: LoginFormProps) {
  const id = useId()
  const emailId = `${id}-email`
  const passwordId = `${id}-password`
  const { login } = useAuth()

  const [values, setValues] = useState<LoginInput>(emptyForm)
  const [errors, setErrors] = useState<LoginErrors>({})
  const [formError, setFormError] = useState("")
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [showPassword, setShowPassword] = useState(false)

  function updateField(field: LoginField) {
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

    const nextErrors = validateLogin(values)
    setErrors(nextErrors)
    setFormError("")
    if (Object.keys(nextErrors).length > 0) {
      return
    }

    setIsSubmitting(true)
    try {
      await login({
        email: values.email.trim().toLowerCase(),
        password: values.password,
      })
      setIsSubmitting(false)
      onSuccess()
    } catch (error) {
      setFormError(loginErrorMessage(error))
      setIsSubmitting(false)
    }
  }

  return (
    <form
      onSubmit={handleSubmit}
      noValidate
      aria-busy={isSubmitting}
      className="flex flex-col gap-6"
    >
      <FieldGroup>
        <Field data-invalid={Boolean(errors.email)}>
          <FieldLabel htmlFor={emailId}>Email</FieldLabel>
          <Input
            id={emailId}
            name="email"
            type="email"
            inputMode="email"
            autoComplete="email"
            autoFocus
            placeholder="ada@example.com"
            className="h-10"
            value={values.email}
            onChange={updateField("email")}
            disabled={isSubmitting}
            aria-invalid={Boolean(errors.email)}
            aria-describedby={errors.email ? `${id}-email-error` : undefined}
          />
          <FieldError id={`${id}-email-error`}>{errors.email}</FieldError>
        </Field>

        <Field data-invalid={Boolean(errors.password)}>
          <FieldLabel htmlFor={passwordId}>Password</FieldLabel>
          <div className="relative">
            <Input
              id={passwordId}
              name="password"
              type={showPassword ? "text" : "password"}
              autoComplete="current-password"
              placeholder="Your password"
              className="h-10 pr-10"
              value={values.password}
              onChange={updateField("password")}
              disabled={isSubmitting}
              aria-invalid={Boolean(errors.password)}
              aria-describedby={
                errors.password ? `${id}-password-error` : undefined
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
          <FieldError id={`${id}-password-error`}>{errors.password}</FieldError>
        </Field>
      </FieldGroup>

      {formError ? (
        <Alert variant="destructive">
          <CircleAlertIcon aria-hidden="true" />
          <AlertDescription>{formError}</AlertDescription>
        </Alert>
      ) : null}

      <Button
        type="submit"
        size="lg"
        className="h-10 w-full"
        disabled={isSubmitting}
      >
        {isSubmitting ? (
          <>
            <Spinner data-icon="inline-start" />
            Signing in...
          </>
        ) : (
          "Sign in"
        )}
      </Button>
    </form>
  )
}
