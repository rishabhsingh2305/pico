import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react"
import { Navigate, Outlet } from "react-router"

import { Spinner } from "@/components/ui/spinner"
import {
  getSession as fetchSession,
  login as loginRequest,
  logout as logoutRequest,
} from "@/features/auth/api"
import { sessionFromUser, type Session } from "@/features/auth/session"
import type { LoginInput } from "@/features/auth/types"

type AuthStatus = "loading" | "ready"

type AuthContextValue = {
  session: Session | null
  status: AuthStatus
  login: (input: LoginInput) => Promise<void>
  logout: () => Promise<void>
  refresh: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

type AuthProviderProps = {
  children: ReactNode
  /** Bootstrap override for tests and future server rendering. */
  initialSession?: Session | null
}

export function AuthProvider({ children, initialSession }: AuthProviderProps) {
  const [session, setSession] = useState<Session | null>(initialSession ?? null)
  const [status, setStatus] = useState<AuthStatus>(
    initialSession === undefined ? "loading" : "ready"
  )

  const refresh = useCallback(async () => {
    try {
      const response = await fetchSession()
      setSession(sessionFromUser(response.user))
    } catch {
      setSession(null)
    } finally {
      setStatus("ready")
    }
  }, [])

  useEffect(() => {
    if (initialSession !== undefined) {
      return
    }

    let cancelled = false
    fetchSession()
      .then((response) => {
        if (!cancelled) {
          setSession(sessionFromUser(response.user))
        }
      })
      .catch(() => {
        if (!cancelled) {
          setSession(null)
        }
      })
      .finally(() => {
        if (!cancelled) {
          setStatus("ready")
        }
      })

    return () => {
      cancelled = true
    }
  }, [initialSession])

  const login = useCallback(async (input: LoginInput) => {
    const response = await loginRequest(input)
    setSession(sessionFromUser(response.user))
    setStatus("ready")
  }, [])

  const logout = useCallback(async () => {
    try {
      await logoutRequest()
    } catch {
      // The cookie may already be gone; clear local state regardless.
    }
    setSession(null)
  }, [])

  const value = useMemo(
    () => ({ session, status, login, logout, refresh }),
    [session, status, login, logout, refresh]
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const value = useContext(AuthContext)
  if (!value) {
    throw new Error("useAuth must be used within AuthProvider")
  }
  return value
}

export function useSession(): Session {
  const { session } = useAuth()
  if (!session) {
    throw new Error("useSession requires an authenticated route")
  }
  return session
}

export function RequireSession() {
  const { status, session } = useAuth()

  if (status === "loading") {
    return <AuthLoading />
  }
  if (!session) {
    return <Navigate to="/login" replace />
  }
  return <Outlet />
}

export function AuthLoading() {
  return (
    <div className="flex min-h-svh items-center justify-center">
      <Spinner className="size-6 text-muted-foreground" />
    </div>
  )
}
