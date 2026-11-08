import { Navigate, Route, Routes } from "react-router"

import { RequireSession } from "@/features/auth/auth-provider"
import { LoginPage } from "@/features/auth/pages/login-page"
import { SignUpPage } from "@/features/auth/pages/sign-up-page"
import { VerifyEmailPage } from "@/features/auth/pages/verify-email-page"
import { CategoryPage } from "@/features/categories/pages/category-page"
import { ExpensesProvider } from "@/features/expenses/expenses-provider"
import { ExpensesPage } from "@/features/expenses/pages/expenses-page"

export function AppRouter() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/signup" element={<SignUpPage />} />
      <Route path="/verify-email" element={<VerifyEmailPage />} />
      <Route element={<RequireSession />}>
        <Route element={<ExpensesProvider />}>
          <Route path="/" element={<ExpensesPage />} />
          <Route path="/categories/:categoryId" element={<CategoryPage />} />
        </Route>
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
