import { Link, Navigate, useNavigate, useSearchParams } from "react-router"

import { AuthLoading, useAuth } from "@/features/auth/auth-provider"
import { AuthLayout } from "@/features/auth/components/auth-layout"
import { LoginForm } from "@/features/auth/components/login-form"
import { usePageTitle } from "@/hooks/use-page-title"

export function LoginPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const email = searchParams.get("email")?.trim() ?? ""
  const { session, status } = useAuth()
  usePageTitle("Sign in · Expenses")

  if (status === "loading") {
    return <AuthLoading />
  }
  if (session) {
    return <Navigate to="/" replace />
  }

  return (
    <AuthLayout
      title="Welcome back"
      description="Sign in to manage your expenses."
      footer={
        <p>
          New here?{" "}
          <Link
            to="/signup"
            className="font-medium text-foreground underline decoration-primary decoration-2 underline-offset-4"
          >
            Create an account
          </Link>
        </p>
      }
    >
      <LoginForm
        initialEmail={email}
        onSuccess={() => navigate("/", { replace: true })}
      />
    </AuthLayout>
  )
}
