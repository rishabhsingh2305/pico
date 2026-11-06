import { AppRouter } from "@/app/router"
import { AuthProvider } from "@/features/auth/auth-provider"

export function App() {
  return (
    <AuthProvider>
      <AppRouter />
    </AuthProvider>
  )
}

export default App
