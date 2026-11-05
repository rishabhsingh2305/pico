import { useState, type ComponentType, type ReactNode } from "react"
import { Link, useLocation } from "react-router"
import { BookOpenIcon, Building2Icon, MenuIcon } from "lucide-react"

import { Brand } from "@/components/brand"
import { Button } from "@/components/ui/button"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet"
import type { TemporarySession } from "@/features/auth/session"
import { cn } from "cn"

type AppShellProps = {
  session: TemporarySession
  children: ReactNode
}

type NavigationItem = {
  to: string
  label: string
  index: string
  icon: ComponentType<{ className?: string }>
  isActive: (pathname: string) => boolean
}

const navigationGroups: { label: string; items: NavigationItem[] }[] = [
  {
    label: "Workspace",
    items: [
      {
        to: "/orgs",
        label: "Organizations",
        index: "01",
        icon: Building2Icon,
        isActive: (pathname) => pathname.startsWith("/orgs"),
      },
    ],
  },
  {
    label: "Resources",
    items: [
      {
        to: "/api-docs",
        label: "API docs",
        index: "02",
        icon: BookOpenIcon,
        isActive: (pathname) => pathname === "/api-docs",
      },
    ],
  },
]

export function AppShell({ session, children }: AppShellProps) {
  const location = useLocation()
  const [mobileOpen, setMobileOpen] = useState(false)

  return (
    <div className="min-h-svh bg-background">
      <aside className="fixed inset-y-0 left-0 z-40 hidden w-[17rem] flex-col border-r border-sidebar-border bg-sidebar text-sidebar-foreground lg:flex">
        <div className="flex h-20 items-center border-b border-sidebar-border px-6">
          <Link
            to="/"
            aria-label="Pico home"
            className="rounded-sm focus-visible:ring-3 focus-visible:ring-sidebar-ring/50 focus-visible:outline-none"
          >
            <Brand align="start" />
          </Link>
        </div>

        <nav aria-label="Workspace navigation" className="flex-1 p-4">
          <NavigationGroups
            pathname={location.pathname}
            onNavigate={() => setMobileOpen(false)}
          />
        </nav>

        <SidebarSession session={session} />
      </aside>

      <div className="lg:pl-[17rem]">
        <header className="sticky top-0 z-30 flex h-16 items-center justify-between border-b border-sidebar-border bg-sidebar px-4 text-sidebar-foreground lg:hidden">
          <Link
            to="/"
            aria-label="Pico home"
            className="rounded-sm focus-visible:ring-3 focus-visible:ring-sidebar-ring/50 focus-visible:outline-none"
          >
            <Brand align="start" />
          </Link>

          <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
            <SheetTrigger
              render={
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  aria-label="Open navigation"
                  className="text-sidebar-foreground hover:bg-sidebar-accent hover:text-sidebar-foreground"
                />
              }
            >
              <MenuIcon aria-hidden="true" />
            </SheetTrigger>
            <SheetContent
              side="left"
              className="w-80 border-sidebar-border bg-sidebar p-0 text-sidebar-foreground"
            >
              <SheetHeader className="border-b border-sidebar-border">
                <SheetTitle className="text-sidebar-foreground">
                  Workspace
                </SheetTitle>
                <SheetDescription className="text-sidebar-foreground/60">
                  Navigate Pico and review your temporary session.
                </SheetDescription>
              </SheetHeader>
              <nav
                aria-label="Workspace navigation"
                className="flex-1 px-3 py-2"
              >
                <NavigationGroups
                  pathname={location.pathname}
                  onNavigate={() => setMobileOpen(false)}
                />
              </nav>
              <SheetFooter className="border-t border-sidebar-border">
                <SessionSummary session={session} />
              </SheetFooter>
            </SheetContent>
          </Sheet>
        </header>

        <main className="mx-auto w-full max-w-[1400px] px-4 py-10 sm:px-6 lg:px-12 lg:py-14">
          {children}
        </main>
      </div>
    </div>
  )
}

function NavigationGroups({
  pathname,
  onNavigate,
}: {
  pathname: string
  onNavigate: () => void
}) {
  return (
    <div className="flex flex-col gap-7">
      {navigationGroups.map((group) => (
        <div key={group.label}>
          <p className="px-3 pb-2 text-label text-sidebar-foreground/45 uppercase">
            {group.label}
          </p>
          <ul className="flex flex-col gap-1">
            {group.items.map((item) => (
              <li key={item.to}>
                <NavigationLink
                  item={item}
                  pathname={pathname}
                  onClick={onNavigate}
                />
              </li>
            ))}
          </ul>
        </div>
      ))}
    </div>
  )
}

function NavigationLink({
  item,
  pathname,
  onClick,
}: {
  item: NavigationItem
  pathname: string
  onClick: () => void
}) {
  const active = item.isActive(pathname)
  const Icon = item.icon

  return (
    <Link
      to={item.to}
      onClick={onClick}
      aria-current={active ? "page" : undefined}
      className={cn(
        "flex items-center gap-3 rounded-md px-3 py-2.5 text-sm font-medium transition-colors focus-visible:ring-3 focus-visible:ring-sidebar-ring/50 focus-visible:outline-none",
        active
          ? "bg-sidebar-primary text-sidebar-primary-foreground"
          : "text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-foreground"
      )}
    >
      <span
        className={cn(
          "font-mono text-[10px] tabular-nums",
          active ? "opacity-70" : "opacity-45"
        )}
        aria-hidden="true"
      >
        {item.index}
      </span>
      <Icon className="size-4" />
      {item.label}
    </Link>
  )
}

function SidebarSession({ session }: { session: TemporarySession }) {
  return (
    <div className="border-t border-sidebar-border p-3">
      <SessionSummary session={session} />
    </div>
  )
}

function SessionSummary({ session }: { session: TemporarySession }) {
  const initial = session.email.trim().charAt(0).toUpperCase() || "P"

  return (
    <div className="flex items-center gap-2.5 rounded-md border border-sidebar-border bg-sidebar-accent/60 p-2.5">
      <div className="flex size-8 shrink-0 items-center justify-center rounded-full bg-sidebar-primary text-xs font-semibold text-sidebar-primary-foreground">
        {initial}
      </div>
      <div className="min-w-0">
        <p className="truncate text-xs font-medium">{session.email}</p>
        <p
          className="truncate text-[10px] tracking-wide text-sidebar-foreground/50 uppercase"
          title="Temporary identity. This is not authentication."
        >
          Temporary session · {session.userId.slice(0, 8)}…
        </p>
      </div>
    </div>
  )
}
