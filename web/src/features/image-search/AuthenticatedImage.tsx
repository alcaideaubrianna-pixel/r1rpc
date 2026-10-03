import { useEffect, useRef, useState } from 'react'
import { Dialog, Flex, Spinner, Text } from '@radix-ui/themes'
import { MagnifyingGlassIcon } from '@radix-ui/react-icons'
import { getBlob } from '../../api/client'

interface AuthenticatedImageProps {
  url: string
  alt: string
  className?: string
  previewTitle?: string
}

export function AuthenticatedImage({ url, alt, className, previewTitle }: AuthenticatedImageProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [visible, setVisible] = useState(false)
  const [source, setSource] = useState('')
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    const element = containerRef.current
    if (!element || typeof IntersectionObserver === 'undefined') {
      setVisible(true)
      return
    }
    const observer = new IntersectionObserver(([entry]) => {
      if (!entry.isIntersecting) return
      setVisible(true)
      observer.disconnect()
    }, { rootMargin: '300px 0px' })
    observer.observe(element)
    return () => observer.disconnect()
  }, [url])

  useEffect(() => {
    if (!visible) return
    let active = true
    let objectURL = ''
    const controller = new AbortController()
    setSource('')
    setFailed(false)
    getBlob(url, controller.signal)
      .then((blob) => {
        if (!active) return
        objectURL = URL.createObjectURL(blob)
        setSource(objectURL)
      })
      .catch((error: unknown) => {
        if (active && !(error instanceof DOMException && error.name === 'AbortError')) setFailed(true)
      })
    return () => {
      active = false
      controller.abort()
      if (objectURL) URL.revokeObjectURL(objectURL)
    }
  }, [url, visible])

  if (failed) return <div ref={containerRef} className={`${className ?? ''} authenticated-image-state`}><Text size="1" color="red">图片加载失败</Text></div>
  if (!source) return <div ref={containerRef} className={`${className ?? ''} authenticated-image-state`}>{visible && <Spinner size="1" />}</div>

  return <Dialog.Root>
    <Dialog.Trigger>
      <button type="button" className={`image-preview-trigger ${className ?? ''}`} aria-label={`预览${alt}`}>
        <img src={source} alt={alt} loading="lazy" decoding="async" />
        <span className="image-preview-hint"><MagnifyingGlassIcon /> 预览</span>
      </button>
    </Dialog.Trigger>
    <Dialog.Content className="image-preview-dialog" maxWidth="min(92vw, 1080px)">
      <Dialog.Title>{previewTitle || alt}</Dialog.Title>
      <Dialog.Description size="2" color="gray">点击遮罩或按 Esc 关闭预览</Dialog.Description>
      <Flex justify="center" mt="4" className="image-preview-stage">
        <img src={source} alt={alt} decoding="async" />
      </Flex>
    </Dialog.Content>
  </Dialog.Root>
}
