import { useEffect, useState } from 'react'

export function useSourceImagePreview() {
  const [sourceImage, setSourceImage] = useState<File | null>(null)
  const [sourcePreview, setSourcePreview] = useState('')
  const [sourceSize, setSourceSize] = useState('')

  useEffect(() => {
    return () => {
      if (sourcePreview) {
        URL.revokeObjectURL(sourcePreview)
      }
    }
  }, [sourcePreview])

  function selectSourceImage(file: File | null) {
    setSourceImage(file)
    setSourceSize('')

    if (!file) {
      setSourcePreview('')
      return
    }

    const previewURL = URL.createObjectURL(file)
    setSourcePreview(previewURL)

    const image = new Image()
    image.onload = () => {
      setSourceSize(`${image.naturalWidth}x${image.naturalHeight}`)
    }
    image.src = previewURL
  }

  return {
    sourceImage,
    sourcePreview,
    sourceSize,
    selectSourceImage,
  }
}
