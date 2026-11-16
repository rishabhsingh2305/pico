import type { User } from "@/features/auth/types"

export type Session = {
  userId: string
  name: string
  email: string
  orgs: Record<string, string[]>
}

export function sessionFromUser(
  user: User,
  orgs: Record<string, string[]> = {}
): Session {
  return {
    userId: user.id,
    name: user.name,
    email: user.email,
    orgs,
  }
}
