import type { User } from "@/features/auth/types"

export type Session = {
  userId: string
  name: string
  email: string
}

export function sessionFromUser(user: User): Session {
  return {
    userId: user.id,
    name: user.name,
    email: user.email,
  }
}
