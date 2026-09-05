const imageReferencePattern = /@([0-9]+)/g

function isWordCharacter(value: string | undefined) {
  return value !== undefined && /[A-Za-z0-9_]/.test(value)
}

export function getImageReferenceNumbers(prompt: string) {
  const references: number[] = []
  for (const match of prompt.matchAll(imageReferencePattern)) {
    const start = match.index ?? 0
    const end = start + match[0].length
    const previous = prompt[start - 1]
    const next = prompt[end]
    if (isWordCharacter(previous) || isWordCharacter(next)) continue

    const number = Number(match[1])
    if (Number.isSafeInteger(number)) references.push(number)
  }
  return references
}

export function validateImageReferences(prompt: string, imageCount: number) {
  for (const number of getImageReferenceNumbers(prompt)) {
    if (number < 1 || number > imageCount) {
      return `图片引用 @${number} 无效，请使用 @1 到 @${imageCount}。`
    }
  }
  return ''
}

export function findImageMention(prompt: string, cursor: number) {
  const beforeCursor = prompt.slice(0, cursor)
  const match = beforeCursor.match(/(^|[^A-Za-z0-9_])@([0-9]*)$/)
  if (!match) return null

  const start = cursor - match[0].length + match[1].length
  return { start, end: cursor, query: match[2] }
}
