import { useState } from "react"

import { CardContent, CardDescription, CardHeader } from "@/components/ui/card"
import { AuthShell } from "@/features/auth/components/auth-shell"
import { SignUpForm } from "@/features/auth/components/sign-up-form"
import { SignUpSuccess } from "@/features/auth/components/sign-up-success"
import type { User } from "@/features/auth/types"
import { usePageTitle } from "@/hooks/use-page-title"

export function SignUpPage() {
  const [createdUser, setCreatedUser] = useState<User | null>(null)
  usePageTitle("Create your account · Pico")

  return (
    <AuthShell footer={<p>Check your inbox to verify your email.</p>}>
      {createdUser ? (
        <CardContent>
          <SignUpSuccess
            email={createdUser.email}
            onReset={() => setCreatedUser(null)}
          />
        </CardContent>
      ) : (
        <>
          <CardHeader className="gap-1.5">
            <h1 className="font-heading text-2xl leading-snug font-semibold tracking-tight">
              Create your account
            </h1>
            <CardDescription>
              Start tracking shared expenses with your team.
            </CardDescription>
          </CardHeader>
          <CardContent className="pt-2">
            <SignUpForm onSuccess={setCreatedUser} />
          </CardContent>
        </>
      )}
    </AuthShell>
  )
}
