import { useEffect, useState } from "react"
import { useLocation, useNavigate } from "react-router"
import { CircleAlertIcon, CheckIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { CardContent } from "@/components/ui/card"
import { Spinner } from "@/components/ui/spinner"
import { AuthShell } from "@/features/auth/components/auth-shell"
import { verifyEmail } from "@/features/auth/api"
import { usePageTitle } from "@/hooks/use-page-title"
import { ApiError } from "@/lib/api-client"

type VerificationStatus =
  "loading" | "success" | "invalid" | "network" | "missing"

const statusCopy: Record<
  Exclude<VerificationStatus, "loading">,
  { title: string; description: string }
> = {
  success: {
    title: "Email verified",
    description:
      "Your email address is confirmed. Sign in to create or open an organization.",
  },
  invalid: {
    title: "Invalid verification link",
    description: "This verification link is invalid or has expired.",
  },
  network: {
    title: "Could not verify your email",
    description:
      "We could not reach the server. Check your connection and try again.",
  },
  missing: {
    title: "Verification link incomplete",
    description:
      "This link is missing its verification token. Open the link from your email again.",
  },
}

export function VerifyEmailPage() {
  const location = useLocation()
  const navigate = useNavigate()
  const token =
    new URLSearchParams(location.hash.replace(/^#/, "")).get("token")?.trim() ??
    ""

  const [status, setStatus] = useState<VerificationStatus>(
    token ? "loading" : "missing"
  )
  const [attempt, setAttempt] = useState(0)

  useEffect(() => {
    if (!token) {
      return
    }

    let cancelled = false

    verifyEmail(token)
      .then(() => {
        if (!cancelled) {
          setStatus("success")
        }
      })
      .catch((error: unknown) => {
        if (cancelled) {
          return
        }
        if (error instanceof ApiError && error.status === 400) {
          setStatus("invalid")
          return
        }
        setStatus("network")
      })

    return () => {
      cancelled = true
    }
  }, [token, attempt])

  function retry() {
    setStatus("loading")
    setAttempt((current) => current + 1)
  }

  const copy = status === "loading" ? null : statusCopy[status]
  usePageTitle(
    status === "success"
      ? "Email verified · Pico"
      : status === "loading"
        ? "Verifying your email · Pico"
        : "Email verification · Pico"
  )

  return (
    <AuthShell>
      <CardContent>
        <div className="flex flex-col items-center gap-5 py-4 text-center">
          {status === "loading" ? (
            <Spinner className="size-6 text-muted-foreground" />
          ) : status === "success" ? (
            <div className="flex size-12 items-center justify-center rounded-full bg-primary/10 text-primary">
              <CheckIcon className="size-6" aria-hidden="true" />
            </div>
          ) : (
            <div className="flex size-12 items-center justify-center rounded-full bg-destructive/10 text-destructive">
              <CircleAlertIcon className="size-6" aria-hidden="true" />
            </div>
          )}

          <div className="flex flex-col gap-2">
            <h1 className="font-heading text-xl font-semibold">
              {copy?.title ?? "Verifying your email"}
            </h1>
            <p className="text-sm text-muted-foreground">
              {copy?.description ?? "This will only take a moment."}
            </p>
          </div>

          {status === "success" ? (
            <Button
              type="button"
              className="w-full"
              onClick={() => navigate("/login")}
            >
              Continue to sign in
            </Button>
          ) : null}

          {status === "network" ? (
            <div className="flex w-full flex-col gap-2">
              <Button type="button" className="w-full" onClick={retry}>
                Try again
              </Button>
              <Button
                type="button"
                variant="outline"
                className="w-full"
                onClick={() => navigate("/signup")}
              >
                Back to sign up
              </Button>
            </div>
          ) : null}

          {status === "invalid" || status === "missing" ? (
            <Button
              type="button"
              variant="outline"
              className="w-full"
              onClick={() => navigate("/signup")}
            >
              Back to sign up
            </Button>
          ) : null}
        </div>
      </CardContent>
    </AuthShell>
  )
}
