import { Maybe } from "@/utils";
import { CustomError, UserDonationData } from ".";

export type AffiliateCode = string | "none";

// Interface to construct request body for /wallet/connection
export interface WalletAffiliateRequest {
  code: AffiliateCode;
  address: string;
}

export interface WalletAffiliateResponse {
  data: Maybe<UserDonationData>;
  error: Maybe<CustomError>;
}

export interface GenAffiliateRequest {
  address: string;
  timestamp: number;
  signature: string;
}

// Interface for /affliate/code response, return generated code to user
export interface GenAffiliateResponse {
  address: string;
  code: AffiliateCode;
  ts: string;
}
