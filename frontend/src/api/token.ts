let accessToken: string | null = null

export function setApiAccessToken(token: string | null) {
  accessToken = token
}

export function getApiAccessToken(): string | null {
  return accessToken
}
