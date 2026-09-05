import { useEffect, useRef, useState } from 'react'

import type { SourceImage } from '../types/image'

export const maxSourceImages = 4

export function useSourceImagePreview() {
  const [sourceImages, setSourceImages] = useState<SourceImage[]>([])
  const sourceImagesRef = useRef<SourceImage[]>([])

  useEffect(() => {
    return () => {
      for (const image of sourceImagesRef.current) URL.revokeObjectURL(image.preview)
    }
  }, [])

  function createSourceImage(file: File): SourceImage {
    const preview = URL.createObjectURL(file)
    const image: SourceImage = {
      id: `${file.name}-${file.lastModified}-${Math.random().toString(36).slice(2)}`,
      file,
      preview,
      size: '',
    }

    const nativeImage = new Image()
    nativeImage.onload = () => {
      setSourceImages((current) => current.map((item) => item.id === image.id
        ? { ...item, size: `${nativeImage.naturalWidth}x${nativeImage.naturalHeight}` }
        : item))
    }
    nativeImage.src = preview
    return image
  }

  function updateSourceImages(nextImages: SourceImage[]) {
    const nextIDs = new Set(nextImages.map((image) => image.id))
    for (const image of sourceImagesRef.current) {
      if (!nextIDs.has(image.id)) URL.revokeObjectURL(image.preview)
    }
    sourceImagesRef.current = nextImages
    setSourceImages(nextImages)
  }

  function selectMainImage(file: File | null) {
    if (!file) return
    const nextImages = [createSourceImage(file), ...sourceImagesRef.current.slice(1)]
    updateSourceImages(nextImages)
  }

  function addReferenceImages(files: File[]) {
    const remaining = maxSourceImages - sourceImagesRef.current.length
    if (remaining <= 0) return
    const nextImages = [
      ...sourceImagesRef.current,
      ...files.slice(0, remaining).map(createSourceImage),
    ]
    updateSourceImages(nextImages)
  }

  function removeSourceImage(id: string) {
    updateSourceImages(sourceImagesRef.current.filter((image) => image.id !== id))
  }

  return {
    sourceImages,
    selectMainImage,
    addReferenceImages,
    removeSourceImage,
  }
}
