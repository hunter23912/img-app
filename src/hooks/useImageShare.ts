import { useState } from 'react'

import { imageURLToFile } from '../utils/image'

export function useImageShare(onMessage: (message: string) => void) {
  const [isSharing, setIsSharing] = useState(false)

  async function shareImage(resultImage: string) {
    if (!resultImage) return

    setIsSharing(true)
    try {
      const file = await imageURLToFile(resultImage, 'gpt-image.png')

      if (navigator.canShare?.({ files: [file] })) {
        await navigator.share({
          files: [file],
          title: '生成图片',
        })
        onMessage('已打开系统分享面板，可以选择保存图片。')
        return
      }

      window.open(resultImage, '_blank', 'noopener,noreferrer')
      onMessage('当前浏览器不支持直接分享图片，已尝试在新页面打开。')
    } catch (error) {
      window.open(resultImage, '_blank', 'noopener,noreferrer')
      onMessage(
        error instanceof Error
          ? `${error.message}。如果 iOS Edge 不能直接存入相册，请长按图片或用分享面板保存。`
          : '如果 iOS Edge 不能直接存入相册，请长按图片或用分享面板保存。'
      )
    } finally {
      setIsSharing(false)
    }
  }

  return {
    isSharing,
    shareImage,
  }
}
