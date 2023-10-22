export type AffiliateCode = string | "none";

// Interface to construct request body for /wallet/connection
export interface WalletAffliateRequest {
  code: AffiliateCode;
  address: string;
}

// Interface for /affliate/code response, return generated code to user
export interface GenAffliateResponse {
  address: string;
  code: AffiliateCode;
  ts: string;
}
