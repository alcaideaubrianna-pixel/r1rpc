import { Badge, Flex, Text } from '@radix-ui/themes'
import type { SearchCandidate } from './types'
import { AuthenticatedImage } from './AuthenticatedImage'

export function CandidateGallery({ candidate }: { candidate: SearchCandidate }) {
  const images = candidate.images ?? []
  const best = images.find((image) => image.imageUrl) ?? images[0]
  return <div className="candidate-gallery">
    <div className="candidate-main-image">
      {best?.imageUrl ? <AuthenticatedImage url={best.imageUrl} alt="最高相似度候选图" previewTitle={`${candidate.title || candidate.contentId} · 图片 #${best.imageIndex + 1}`} /> : <span>等待下载</span>}
      {best?.downloadStatus === 'analyzed' && <Badge className="candidate-best-badge" color={best.matched ? 'green' : 'gray'}>{(best.score * 100).toFixed(1)}%</Badge>}
    </div>
    {images.length > 1 && <Flex gap="2" mt="2" wrap="wrap">
      {images.filter((image) => image.imageIndex !== best?.imageIndex).map((image) => <div key={image.imageIndex} className="candidate-thumb">
        {image.imageUrl ? <AuthenticatedImage url={image.imageUrl} alt={`候选图 ${image.imageIndex + 1}`} previewTitle={`${candidate.title || candidate.contentId} · 图片 #${image.imageIndex + 1}`} /> : <Text size="1" color={image.downloadStatus === 'failed' ? 'red' : 'gray'}>#{image.imageIndex + 1}</Text>}
      </div>)}
    </Flex>}
  </div>
}
