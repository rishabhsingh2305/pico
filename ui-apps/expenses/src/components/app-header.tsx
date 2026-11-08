import { useState } from "react"
import { useNavigate } from "react-router"
import { LogOutIcon } from "lucide-react"

import { Brand } from "@/components/brand"
import { Button } from "@/components/ui/button"
import { Spinner } from "@/components/ui/spinner"
import { useAuth } from "@/features/auth/auth-provider"

export function AppHeader() {
  const { logout } = useAuth()
  const navigate = useNavigate()
  const [isSigningOut, setIsSigningOut] = useState(false)

  async function handleSignOut() {
    setIsSigningOut(true)
    await logout()
    navigate("/login", { replace: true })
  }

  return (
    <header className="shrink-0 border-b-2 border-foreground bg-background">
      <div className="mx-auto flex h-14 w-full max-w-5xl items-center justify-between px-4 sm:h-16">
        <Brand />
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="h-9"
          onClick={handleSignOut}
          disabled={isSigningOut}
        >
          {isSigningOut ? (
            <Spinner data-icon="inline-start" />
          ) : (
            <LogOutIcon data-icon="inline-start" aria-hidden="true" />
          )}
          Sign out
        </Button>
      </div>
    </header>
  )
}
