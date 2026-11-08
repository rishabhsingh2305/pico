import { useState, type FormEvent } from "react"
import { Link, useNavigate, useSearchParams } from "react-router"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Spinner } from "@/components/ui/spinner"
import { resendVerification } from "@/features/auth/api"
import { AuthLayout } from "@/features/auth/components/auth-layout"
import { OtpForm } from "@/features/auth/components/otp-form"
import { VerificationSuccess } from "@/features/auth/components/verification-success"
import { resendErrorMessage } from "@/features/auth/error-message"
import { validateEmailAddress } from "@/features/auth/validation"
import { usePageTitle } from "@/hooks/use-page-title"

export function VerifyEmailPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const initialEmail = searchParams.get("email")?.trim().toLowerCase() ?? ""

  const [email, setEmail] = useState(initialEmail)
  const [cooldown, setCooldown] = useState(0)
  const [verified, setVerified] = useState(false)
  usePageTitle("Verify your email · Expenses")

  if (verified) {
    return (
      <AuthLayout
        title="Email verified"
        description="Your account is ready to use."
      >
        <VerificationSuccess
          email={email}
          onContinue={() =>
            navigate(`/login?email=${encodeURIComponent(email)}`, {
              replace: true,
            })
          }
        />
      </AuthLayout>
    )
  }

  if (email) {
    return (
      <AuthLayout
        title="Verify your email"
        description="Enter the 6-digit code we sent to your inbox."
        footer={
          <p>
            <Link
              to="/login"
              className="font-medium text-foreground underline decoration-primary decoration-2 underline-offset-4"
            >
              Back to sign in
            </Link>
          </p>
        }
      >
        <OtpForm
          email={email}
          initialCooldown={cooldown}
          onVerified={() => setVerified(true)}
        />
      </AuthLayout>
    )
  }

  return (
    <RequestCodePage
      onSent={(nextEmail) => {
        setEmail(nextEmail)
        setCooldown(60)
      }}
    />
  )
}

type RequestCodePageProps = {
  onSent: (email: string) => void
}

function RequestCodePage({ onSent }: RequestCodePageProps) {
  const [email, setEmail] = useState("")
  const [error, setError] = useState("")
  const [formError, setFormError] = useState("")
  const [isSubmitting, setIsSubmitting] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (isSubmitting) {
      return
    }

    const nextError = validateEmailAddress(email)
    setError(nextError ?? "")
    setFormError("")
    if (nextError) {
      return
    }

    const normalized = email.trim().toLowerCase()
    setIsSubmitting(true)
    try {
      await resendVerification(normalized)
      setIsSubmitting(false)
      onSent(normalized)
    } catch (caught) {
      setFormError(resendErrorMessage(caught))
      setIsSubmitting(false)
    }
  }

  return (
    <AuthLayout
      title="Verify your email"
      description="We'll send a new 6-digit code to your inbox."
      footer={
        <p>
          <Link
            to="/login"
            className="font-medium text-foreground underline underline-offset-4"
          >
            Back to sign in
          </Link>
        </p>
      }
    >
      <form
        onSubmit={handleSubmit}
        noValidate
        aria-busy={isSubmitting}
        className="flex flex-col gap-5"
      >
        <div className="flex flex-col gap-2">
          <Label htmlFor="verify-email">Email</Label>
          <Input
            id="verify-email"
            name="email"
            type="email"
            inputMode="email"
            autoComplete="email"
            autoFocus
            placeholder="you@example.com"
            className="h-10"
            value={email}
            onChange={(event) => {
              setEmail(event.target.value)
              if (error) {
                setError("")
              }
              if (formError) {
                setFormError("")
              }
            }}
            disabled={isSubmitting}
            aria-invalid={Boolean(error)}
            aria-describedby={error ? "verify-email-error" : undefined}
          />
          {error ? (
            <p id="verify-email-error" className="text-sm text-destructive">
              {error}
            </p>
          ) : null}
        </div>

        {formError ? (
          <Alert variant="destructive">
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
              Sending...
            </>
          ) : (
            "Send code"
          )}
        </Button>
      </form>
    </AuthLayout>
  )
}
