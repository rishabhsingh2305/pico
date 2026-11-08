import { useEffect, useId, useState, type FormEvent } from "react"
import { CircleAlertIcon, MailCheckIcon } from "lucide-react"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Spinner } from "@/components/ui/spinner"
import { resendVerification, verifyEmailOtp } from "@/features/auth/api"
import {
  otpErrorMessage,
  resendErrorMessage,
} from "@/features/auth/error-message"
import { validateOtp } from "@/features/auth/validation"

type OtpFormProps = {
  email: string
  initialCooldown?: number
  onVerified: () => void
}

export function OtpForm({
  email,
  initialCooldown = 0,
  onVerified,
}: OtpFormProps) {
  const id = useId()
  const codeId = `${id}-code`

  const [code, setCode] = useState("")
  const [error, setError] = useState("")
  const [formError, setFormError] = useState("")
  const [notice, setNotice] = useState("")
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [isResending, setIsResending] = useState(false)
  const [cooldown, setCooldown] = useState(initialCooldown)

  useEffect(() => {
    if (cooldown <= 0) {
      return
    }
    const timer = window.setTimeout(() => {
      setCooldown((current) => Math.max(0, current - 1))
    }, 1000)
    return () => window.clearTimeout(timer)
  }, [cooldown])

  function handleCodeChange(value: string) {
    setCode(value.replace(/\D/g, "").slice(0, 6))
    if (error) {
      setError("")
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

    const nextError = validateOtp(code)
    setError(nextError ?? "")
    setFormError("")
    if (nextError) {
      return
    }

    setIsSubmitting(true)
    try {
      await verifyEmailOtp(email, code)
      setIsSubmitting(false)
      onVerified()
    } catch (caught) {
      setFormError(otpErrorMessage(caught))
      setIsSubmitting(false)
    }
  }

  async function handleResend() {
    if (isResending || cooldown > 0) {
      return
    }

    setIsResending(true)
    setFormError("")
    setNotice("")
    try {
      const response = await resendVerification(email)
      setNotice("A new code is on its way.")
      setCooldown(
        response.retryAfterSeconds > 0 ? response.retryAfterSeconds : 60
      )
    } catch (caught) {
      setFormError(resendErrorMessage(caught))
    } finally {
      setIsResending(false)
    }
  }

  return (
    <form
      onSubmit={handleSubmit}
      noValidate
      aria-busy={isSubmitting}
      className="flex flex-col gap-5"
    >
      <div className="flex items-center gap-3 border-2 border-foreground bg-muted/40 p-3">
        <MailCheckIcon
          className="size-5 shrink-0 text-muted-foreground"
          aria-hidden="true"
        />
        <p className="text-sm text-muted-foreground">
          Code sent to{" "}
          <span className="font-medium text-foreground">{email}</span>
        </p>
      </div>

      <div className="flex flex-col gap-2">
        <Label htmlFor={codeId}>Verification code</Label>
        <Input
          id={codeId}
          name="code"
          inputMode="numeric"
          autoComplete="one-time-code"
          autoFocus
          maxLength={6}
          placeholder="000000"
          className="h-12 text-center font-display text-xl tracking-[0.35em]"
          value={code}
          onChange={(event) => handleCodeChange(event.target.value)}
          disabled={isSubmitting}
          aria-invalid={Boolean(error)}
          aria-describedby={error ? `${id}-code-error` : undefined}
        />
        {error ? (
          <p id={`${id}-code-error`} className="text-sm text-destructive">
            {error}
          </p>
        ) : null}
      </div>

      {formError ? (
        <Alert variant="destructive">
          <CircleAlertIcon aria-hidden="true" />
          <AlertDescription>{formError}</AlertDescription>
        </Alert>
      ) : null}

      {notice ? (
        <p role="status" className="text-sm text-muted-foreground">
          {notice}
        </p>
      ) : null}

      <Button
        type="submit"
        size="lg"
        className="h-11 w-full"
        disabled={isSubmitting || code.length !== 6}
      >
        {isSubmitting ? (
          <>
            <Spinner data-icon="inline-start" />
            Verifying...
          </>
        ) : (
          "Verify email"
        )}
      </Button>

      <Button
        type="button"
        variant="ghost"
        className="w-full"
        onClick={handleResend}
        disabled={isResending || cooldown > 0}
      >
        {isResending ? (
          <>
            <Spinner data-icon="inline-start" />
            Sending...
          </>
        ) : cooldown > 0 ? (
          `Resend code in ${cooldown}s`
        ) : (
          "Resend code"
        )}
      </Button>
    </form>
  )
}
