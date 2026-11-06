import { Link, Navigate, useNavigate } from "react-router"

import { CardContent, CardDescription, CardHeader } from "@/components/ui/card"
import { AuthLoading, useAuth } from "@/features/auth/auth-provider"
import { AuthShell } from "@/features/auth/components/auth-shell"
import { LoginForm } from "@/features/auth/components/login-form"
import { usePageTitle } from "@/hooks/use-page-title"

export function LoginPage() {
  const navigate = useNavigate()
  const { session, status } = useAuth()
  usePageTitle("Sign in · Pico")

  if (status === "loading") {
    return <AuthLoading />
  }
  if (session) {
    return <Navigate to="/orgs" replace />
  }

  return (
    <AuthShell
      footer={
        <p>
          New to Pico?{" "}
          <Link
            to="/signup"
            className="font-medium text-foreground underline-offset-4 hover:underline"
          >
            Create an account
          </Link>
        </p>
      }
    >
      <CardHeader className="gap-1.5">
        <h1 className="font-heading text-2xl leading-snug font-semibold tracking-tight">
          Sign in
        </h1>
        <CardDescription>
          Continue tracking shared expenses with your team.
        </CardDescription>
      </CardHeader>
      <CardContent className="pt-2">
        <LoginForm onSuccess={() => navigate("/orgs", { replace: true })} />
      </CardContent>
    </AuthShell>
  )
}
