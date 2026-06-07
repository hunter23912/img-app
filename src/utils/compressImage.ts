export type ImageOutputType = 'image/jpeg' | 'image/png'

export type CompressionOutputType = 'auto' | ImageOutputType

export interface CompressImageOptions {
  quality: number
  outputType: ImageOutputType
}

export interface CompressImageResult {
  blob: Blob
  url: string
  width: number
  height: number
}

export async function compressImage(file: File, options: CompressImageOptions) {
  const source = await loadImage(file)
  const width = source.naturalWidth
  const height = source.naturalHeight

  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height

  const context = canvas.getContext('2d')
  if (!context) {
    throw new Error('当前浏览器不支持图片压缩。')
  }

  if (options.outputType === 'image/jpeg') {
    context.fillStyle = '#ffffff'
    context.fillRect(0, 0, width, height)
  }

  context.drawImage(source, 0, 0, width, height)

  const blob = await canvasToBlob(canvas, options.outputType, options.quality)
  return {
    blob,
    url: URL.createObjectURL(blob),
    width,
    height,
  }
}

export function formatBytes(bytes: number) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(2)} MB`
}

export function outputExtension(outputType: ImageOutputType) {
  if (outputType === 'image/jpeg') return 'jpg'
  return 'png'
}

export function extensionFromOutput(output: string) {
  if (output === 'png') return 'png'
  return 'jpg'
}

export function compressionPercent(originalBytes: number, compressedBytes: number) {
  if (originalBytes <= 0) return 0
  return Math.max(0, Math.round((1 - compressedBytes / originalBytes) * 100))
}

function loadImage(file: File) {
  return new Promise<HTMLImageElement>((resolve, reject) => {
    const url = URL.createObjectURL(file)
    const image = new Image()

    image.onload = () => {
      URL.revokeObjectURL(url)
      resolve(image)
    }
    image.onerror = () => {
      URL.revokeObjectURL(url)
      reject(new Error('图片读取失败。'))
    }
    image.src = url
  })
}

function canvasToBlob(canvas: HTMLCanvasElement, type: ImageOutputType, quality: number) {
  return new Promise<Blob>((resolve, reject) => {
    canvas.toBlob(
      (blob) => {
        if (!blob) {
          reject(new Error('图片压缩失败，请尝试其他输出格式。'))
          return
        }

        resolve(blob)
      },
      type,
      quality
    )
  })
}
