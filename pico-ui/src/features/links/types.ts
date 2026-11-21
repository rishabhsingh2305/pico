export type ShortLink = {
  id: string
  orgId: string
  userId: string
  slug: string
  url: string
  title: string
  tags: string[]
  shortPath: string
  shortUrl: string
  disabled: boolean
  createdAt: string
  updatedAt: string
}

export type LinkPage = {
  items: ShortLink[]
  nextOffset: number | null
}

export type LinkStat = {
  linkId: string
  clicks: number
}

export type LinkStatsResponse = {
  items: LinkStat[]
}

export type LinkFilters = {
  q: string
  tags: string[]
}

export type CreateLinkInput = {
  url: string
  slug?: string
  title?: string
  tags: string[]
}

export type UpdateLinkInput = {
  url?: string
  title?: string
  tags?: string[]
  disabled?: boolean
}
