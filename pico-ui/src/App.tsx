import { AppRouter } from "@/app/router"
import { QueryProvider } from "@/app/query-provider"
import { AuthProvider } from "@/features/auth/auth-provider"

export function App() {
  return (
    <QueryProvider>
      <AuthProvider>
        <AppRouter />
      </AuthProvider>
    </QueryProvider>
  )
}

export default App
