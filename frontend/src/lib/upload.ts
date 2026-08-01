/** Display title for an upload: custom/auto name, else filename. */
export function uploadDisplayName(u: {
  name?: string | null
  filename?: string | null
  id?: string
}): string {
  const name = u.name?.trim()
  if (name) return name
  const filename = u.filename?.trim()
  if (filename) return filename
  return u.id ?? 'Upload'
}
