import { Outlet } from "react-router"
import { BookOpenIcon, Building2Icon, KeyRoundIcon } from "lucide-react"

import { AppShell, type NavigationGroup } from "@/components/app-shell"
import { useSession } from "@/features/auth/auth-provider"

const navigation: NavigationGroup[] = [
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
    label: "Developer",
    items: [
      {
        to: "/api",
        label: "API console",
        index: "02",
        icon: KeyRoundIcon,
        isActive: (pathname) => pathname === "/api",
      },
      {
        to: "/api-docs",
        label: "API docs",
        index: "03",
        icon: BookOpenIcon,
        isActive: (pathname) => pathname === "/api-docs",
        newTab: true,
      },
    ],
  },
]

export function WorkspaceLayout() {
  const session = useSession()

  return (
    <AppShell session={session} navigation={navigation}>
      <Outlet />
    </AppShell>
  )
}
