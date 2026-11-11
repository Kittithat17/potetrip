export const MAX_UPLOAD_BYTES = 8 * 1024 * 1024

/**
 * Downscales an image so its longer side is at most maxSide px and re-encodes
 * it as JPEG (this also drops EXIF metadata such as GPS). If the browser can't
 * decode the format (e.g. HEIC outside Safari) the original file is returned
 * and the server handles it.
 */
export async function resizeImage(file: File, maxSide = 1600, quality = 0.85): Promise<Blob> {
  let bitmap: ImageBitmap
  try {
    bitmap = await createImageBitmap(file)
  } catch {
    return file
  }
  try {
    const scale = Math.min(1, maxSide / Math.max(bitmap.width, bitmap.height))
    const w = Math.round(bitmap.width * scale)
    const h = Math.round(bitmap.height * scale)
    const canvas = document.createElement('canvas')
    canvas.width = w
    canvas.height = h
    const ctx = canvas.getContext('2d')
    if (!ctx) return file
    ctx.drawImage(bitmap, 0, 0, w, h)
    const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/jpeg', quality))
    return blob && blob.size < file.size ? blob : file
  } finally {
    bitmap.close()
  }
}

/** Runs fn over items with at most `limit` in flight. */
export async function mapLimit<T>(items: T[], limit: number, fn: (item: T, index: number) => Promise<void>): Promise<void> {
  let next = 0
  const worker = async () => {
    while (next < items.length) {
      const i = next++
      await fn(items[i], i)
    }
  }
  await Promise.all(Array.from({ length: Math.min(limit, items.length) }, worker))
}
