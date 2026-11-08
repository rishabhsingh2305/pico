import { useState } from "react"
import { Link, useNavigate } from "react-router"

import { Button } from "@/components/ui/button"
import { AuthLayout } from "@/features/auth/components/auth-layout"
import { OtpForm } from "@/features/auth/components/otp-form"
import { SignUpForm } from "@/features/auth/components/sign-up-form"
import { VerificationSuccess } from "@/features/auth/components/verification-success"
import { usePageTitle } from "@/hooks/use-page-title"

type Step = "form" | "otp" | "verified"

export function SignUpPage() {
  const navigate = useNavigate()
  const [step, setStep] = useState<Step>("form")
  const [email, setEmail] = useState("")
  usePageTitle("Create your account · Expenses")

  const loginLink = (
    <Link
      to="/login"
      className="font-medium text-foreground underline decoration-primary decoration-2 underline-offset-4"
    >
      Sign in
    </Link>
  )

  if (step === "verified") {
    return (
      <AuthLayout
        title="Email verified"
        description="Your account is ready to use."
        footer={<p>You can now sign in.</p>}
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

  if (step === "otp") {
    return (
      <AuthLayout
        title="Verify your email"
        description="Enter the 6-digit code we sent to your inbox."
        footer={
          <Button
            type="button"
            variant="link"
            className="h-auto p-0 text-muted-foreground"
            onClick={() => setStep("form")}
          >
            Use a different email
          </Button>
        }
      >
        <OtpForm
          email={email}
          initialCooldown={60}
          onVerified={() => setStep("verified")}
        />
      </AuthLayout>
    )
  }

  return (
    <AuthLayout
      title="Create your account"
      description="Start tracking your expenses in one place."
      footer={<p>Already have an account? {loginLink}</p>}
    >
      <SignUpForm
        onSuccess={(user) => {
          setEmail(user.email)
          setStep("otp")
        }}
      />
    </AuthLayout>
  )
}
