export function dataURLToBlob(dataURL: string) {
  const [metadata, content] = dataURL.split(',')
  const mime = metadata.match(/^data:(.*?);base64$/)?.[1] || 'image/png'
  const binary = atob(content)
  const bytes = new Uint8Array(binary.length)

  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index)
  }

  return new Blob([bytes], { type: mime })
}

export async function imageURLToFile(imageURL: string, filename: string) {
  const blob = imageURL.startsWith('data:')
    ? dataURLToBlob(imageURL)
    : await fetch(imageURL).then((response) => {
        if (!response.ok) {
          throw new Error(`图片下载失败：${response.status}`)
        }
        return response.blob()
      })

  return new File([blob], filename, {
    type: blob.type || 'image/png',
  })
}
