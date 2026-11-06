import type { ReactNode } from "react"

import { AuthProvider } from "@/features/auth/auth-provider"
import type { Session } from "@/features/auth/session"

export const testSession: Session = {
  userId: "user-1",
  email: "ada@example.com",
}

export function AuthTestProvider({
  children,
  session = testSession,
}: {
  children: ReactNode
  session?: Session | null
}) {
  return <AuthProvider initialSession={session}>{children}</AuthProvider>
}
