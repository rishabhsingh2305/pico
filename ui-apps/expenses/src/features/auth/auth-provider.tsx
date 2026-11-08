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
import { clearAuthToken, getAuthToken, setAuthToken } from "@/lib/auth-token"

type AuthStatus = "loading" | "ready"

type AuthContextValue = {
  session: Session | null
  status: AuthStatus
  login: (input: LoginInput) => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

type AuthProviderProps = {
  children: ReactNode
}

export function AuthProvider({ children }: AuthProviderProps) {
  const [session, setSession] = useState<Session | null>(null)
  const [status, setStatus] = useState<AuthStatus>(() =>
    getAuthToken() ? "loading" : "ready"
  )

  useEffect(() => {
    if (!getAuthToken()) {
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
          clearAuthToken()
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
  }, [])

  const login = useCallback(async (input: LoginInput) => {
    const response = await loginRequest(input)
    setAuthToken(response.accessToken)
    setSession(sessionFromUser(response.user))
    setStatus("ready")
  }, [])

  const logout = useCallback(async () => {
    clearAuthToken()
    try {
      await logoutRequest()
    } catch {
      // The request may fail; the local token is already cleared.
    }
    setSession(null)
  }, [])

  const value = useMemo(
    () => ({ session, status, login, logout }),
    [session, status, login, logout]
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
