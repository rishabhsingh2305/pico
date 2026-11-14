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
    <header className="shrink-0 border-b border-border bg-background">
      <div className="mx-auto flex h-16 w-full max-w-5xl items-center justify-between px-4 sm:h-[4.5rem]">
        <Brand />
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="h-9 rounded-xl border-border/80 bg-transparent px-3 text-xs font-medium text-muted-foreground hover:bg-muted hover:text-foreground"
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
