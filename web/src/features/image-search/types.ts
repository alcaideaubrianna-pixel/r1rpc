export interface SearchRequestSummary {
  id: string
  externalId?: string
  status: string
  pipelineName: string
  pipelineVersion: string
  groupCount: number
  matchedCount: number
  notMatchedCount: number
  runningCount: number
  failedCount: number
  createdAt: string
}

export interface SearchGroup {
  id: string
  externalId?: string
  subjectUserId?: string
  status: string
  itemCount: number
  queuedCount: number
  runningCount: number
  completedCount: number
  failedCount: number
  bestScore?: number
  bestMatchId?: string
}

export interface SearchItem {
  id: string
  groupId: string
  fileId: string
  fileUrl: string
  ordinal: number
  status: string
  jobId?: string
  errorCode?: string
  errorMessage?: string
}

export interface SourceImage {
  id: string
  groupId: string
  searchItemId?: string
  fileId?: string
  fileUrl?: string
  imageIndex: number
  downloadStatus: string
  preprocessStatus?: string
  filterDecision?: string
  filterReason?: string
}

export interface SearchRequestPage {
  items: SearchRequestSummary[]
  page: number
  pageSize: number
  total: number
}

export interface SearchRequestDetail {
  request: SearchRequestSummary
  groups: SearchGroup[]
  items: SearchItem[]
  sourceImages: SourceImage[]
}

export interface SearchCandidate {
  id: string
  searchItemId: string
  rank: number
  contentId: string
  title?: string
  authorId?: string
  authorName?: string
  coverUrl?: string
  imageFileId?: string
  imageUrl?: string
  downloadStatus: string
  errorMessage?: string
  algorithmVersion?: string
  phashDistance?: number
  dhashDistance?: number
  ahashDistance?: number
  score: number
  matched: boolean
  imageCount: number
  analyzedImages: number
  failedImages: number
	images: CandidateImage[]
}

export interface CandidateImage {
  imageIndex: number
  imageFileId?: string
  imageUrl?: string
  downloadStatus: string
  errorMessage?: string
  score: number
  matched: boolean
  phashDistance?: number
  dhashDistance?: number
  ahashDistance?: number
}

export interface SearchResponseRecord {
  id: string
  jobId: string
  requestId: string
  itemCount: number
  parseStatus: string
  errorMessage?: string
  normalizedJson: string
  rawJson?: string
  createdAt: string
}

export interface SearchCandidatePage {
  items: SearchCandidate[]
  page: number
  pageSize: number
  total: number
}
