import { requestJSON } from "@/lib/api-client"
import type { Category, CreateCategoryInput } from "@/features/categories/types"

const DEFAULT_CURRENCY = "INR"
const CATEGORY_LIST_LIMIT = 100

type ProjectResponse = {
  id: string
  name: string
  currency: string
}

export async function listCategories(orgId: string): Promise<Category[]> {
  const query = new URLSearchParams({
    orgId,
    limit: String(CATEGORY_LIST_LIMIT),
  })
  const response = await requestJSON<{ items: ProjectResponse[] }>(
    `/projects?${query}`,
    { method: "GET" }
  )
  return response.items.map(categoryFromProject)
}

export async function getCategory(
  orgId: string,
  categoryId: string
): Promise<Category> {
  const query = new URLSearchParams({ orgId })
  const project = await requestJSON<ProjectResponse>(
    `/projects/${encodeURIComponent(categoryId)}?${query}`,
    { method: "GET" }
  )
  return categoryFromProject(project)
}

export function createCategory(
  orgId: string,
  input: CreateCategoryInput
): Promise<Category> {
  const query = new URLSearchParams({ orgId })
  return requestJSON<ProjectResponse>(`/projects?${query}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      name: input.name.trim(),
      currency: DEFAULT_CURRENCY,
      tags: [],
    }),
  }).then(categoryFromProject)
}

function categoryFromProject(project: ProjectResponse): Category {
  return {
    id: project.id,
    name: project.name,
    currency: project.currency,
  }
}
