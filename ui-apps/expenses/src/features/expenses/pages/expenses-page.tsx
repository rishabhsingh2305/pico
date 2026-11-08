import { AppShell } from "@/components/app-shell"
import { CategoryList } from "@/features/categories/components/category-list"
import { usePageTitle } from "@/hooks/use-page-title"

export function ExpensesPage() {
  usePageTitle("Expenses")

  return (
    <AppShell className="overflow-y-auto overscroll-contain">
      <main className="mx-auto w-full max-w-5xl px-4 py-5 pb-[calc(1.25rem+env(safe-area-inset-bottom))] sm:py-8">
        <CategoryList />
      </main>
    </AppShell>
  )
}
