import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react"
import { Navigate, Outlet } from "react-router"

import { useAuth } from "@/features/auth/auth-provider"
import {
  CheckingScreen,
  SetupErrorScreen,
  SetupScreen,
} from "@/features/expenses/components/setup-screen"
import { ensureExpensesOrg } from "@/features/expenses/scope"
import { ApiError } from "@/lib/api-client"

type ExpensesContextValue = {
  orgId: string
}

const ExpensesContext = createContext<ExpensesContextValue | null>(null)

type ExpensesState =
  | { status: "checking" }
  | { status: "creating" }
  | { status: "ready"; orgId: string }
  | { status: "error"; message: string }

export function ExpensesProvider() {
  const { session, logout } = useAuth()
  const [state, setState] = useState<ExpensesState>({ status: "checking" })
  const [attempt, setAttempt] = useState(0)

  useEffect(() => {
    if (!session) {
      return
    }

    let cancelled = false

    ensureExpensesOrg(session.userId, () => {
      if (!cancelled) {
        setState({ status: "creating" })
      }
    })
      .then((org) => {
        if (!cancelled) {
          setState({ status: "ready", orgId: org.id })
        }
      })
      .catch(async (error: unknown) => {
        if (cancelled) {
          return
        }
        if (error instanceof ApiError && error.status === 401) {
          await logout()
          return
        }
        setState({ status: "error", message: setupErrorMessage(error) })
      })

    return () => {
      cancelled = true
    }
  }, [session, logout, attempt])

  const retry = useCallback(() => {
    setState({ status: "checking" })
    setAttempt((current) => current + 1)
  }, [])

  const signOut = useCallback(async () => {
    await logout()
  }, [logout])

  if (!session) {
    return <Navigate to="/login" replace />
  }
  if (state.status === "checking") {
    return <CheckingScreen />
  }
  if (state.status === "creating") {
    return <SetupScreen />
  }
  if (state.status === "error") {
    return (
      <SetupErrorScreen
        message={state.message}
        onRetry={retry}
        onSignOut={signOut}
      />
    )
  }

  return (
    <ExpensesContext.Provider value={{ orgId: state.orgId }}>
      <Outlet />
    </ExpensesContext.Provider>
  )
}

export function useExpenses(): ExpensesContextValue {
  const value = useContext(ExpensesContext)
  if (!value) {
    throw new Error("useExpenses must be used within ExpensesProvider")
  }
  return value
}

function setupErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status === 400) {
      return error.message
    }
    return "The service is unavailable right now. Please try again."
  }
  if (error instanceof TypeError) {
    return "Could not reach the server. Check your connection and try again."
  }
  return "Something went wrong while preparing your expense tracker."
}
