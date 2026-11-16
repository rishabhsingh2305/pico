import { requestJSON } from "@/lib/api-client"
import type { CreateProjectInput, Project } from "@/features/projects/types"

const PROJECT_LIST_LIMIT = 100

export async function getProject(
  orgId: string,
  projectId: string
): Promise<Project> {
  const query = new URLSearchParams({ orgId })
  return requestJSON<Project>(
    `/projects/${encodeURIComponent(projectId)}?${query}`,
    { method: "GET" }
  )
}

export async function listProjects(
  orgId: string,
  tags: string[] = []
): Promise<Project[]> {
  const query = new URLSearchParams({
    orgId,
    limit: String(PROJECT_LIST_LIMIT),
  })
  if (tags.length > 0) {
    query.set("tags", tags.join(","))
  }
  const response = await requestJSON<{ items: Project[] }>(
    `/projects?${query}`,
    { method: "GET" }
  )
  return response.items
}

export function createProject(
  orgId: string,
  input: CreateProjectInput
): Promise<Project> {
  const query = new URLSearchParams({ orgId })
  return requestJSON<Project>(`/projects?${query}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  })
}

export function updateProjectTags(
  orgId: string,
  projectId: string,
  tags: string[]
): Promise<Project> {
  const query = new URLSearchParams({ orgId })
  return requestJSON<Project>(
    `/projects/${encodeURIComponent(projectId)}?${query}`,
    {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ tags }),
    }
  )
}
