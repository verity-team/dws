export interface NonceInfo {
  nonce: string;
  expirationTime: string;
  issuedAt: string;
}

export interface VerifyAccessTokenPayload {
  address: string;
  accessToken: string;
}

export interface VerifySignaturePayload {
  message: string;
  signature: string;
}

export interface VerifySignatureResponse {
  accessToken: string;
}
