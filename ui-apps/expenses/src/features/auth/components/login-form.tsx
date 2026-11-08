import { useId, useState, type ChangeEvent, type FormEvent } from "react"
import { Link } from "react-router"
import { CircleAlertIcon, EyeIcon, EyeOffIcon } from "lucide-react"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Spinner } from "@/components/ui/spinner"
import { useAuth } from "@/features/auth/auth-provider"
import { loginErrorMessage } from "@/features/auth/error-message"
import type { LoginInput } from "@/features/auth/types"
import {
  validateLogin,
  type LoginErrors,
  type LoginField,
} from "@/features/auth/validation"
import { ApiError } from "@/lib/api-client"

type LoginFormProps = {
  initialEmail?: string
  onSuccess: () => void
}

export function LoginForm({ initialEmail = "", onSuccess }: LoginFormProps) {
  const id = useId()
  const emailId = `${id}-email`
  const passwordId = `${id}-password`
  const { login } = useAuth()

  const [values, setValues] = useState<LoginInput>(() => ({
    email: initialEmail,
    password: "",
  }))
  const [errors, setErrors] = useState<LoginErrors>({})
  const [formError, setFormError] = useState("")
  const [unverifiedEmail, setUnverifiedEmail] = useState("")
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
      if (unverifiedEmail) {
        setUnverifiedEmail("")
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
    setUnverifiedEmail("")
    if (Object.keys(nextErrors).length > 0) {
      return
    }

    const email = values.email.trim().toLowerCase()
    setIsSubmitting(true)
    try {
      await login({ email, password: values.password })
      setIsSubmitting(false)
      onSuccess()
    } catch (error) {
      if (error instanceof ApiError && error.status === 403) {
        setUnverifiedEmail(email)
      }
      setFormError(loginErrorMessage(error))
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
        <Label htmlFor={emailId}>Email</Label>
        <Input
          id={emailId}
          name="email"
          type="email"
          inputMode="email"
          autoComplete="email"
          autoFocus
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
        {errors.password ? (
          <p id={`${id}-password-error`} className="text-sm text-destructive">
            {errors.password}
          </p>
        ) : null}
      </div>

      {formError ? (
        <Alert variant="destructive">
          <CircleAlertIcon aria-hidden="true" />
          <AlertDescription>
            {formError}
            {unverifiedEmail ? (
              <>
                {" "}
                <Link
                  to={`/verify-email?email=${encodeURIComponent(unverifiedEmail)}`}
                  className="font-medium text-foreground underline decoration-primary decoration-2 underline-offset-4"
                >
                  Verify now
                </Link>
              </>
            ) : null}
          </AlertDescription>
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
            Signing in...
          </>
        ) : (
          "Sign in"
        )}
      </Button>
    </form>
  )
}
