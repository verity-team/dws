export interface FailedResponse {
  code: string;
  message: string;
}

export interface DonationData {
  prices: TokenPrice[];
  stats: DonationStats;
  receiving_address: string;
  status: CampaignStatus;
}

export interface UserDonationData {
  donations: Donation[];
  stats: UserStats;
}

export interface AffiliateDonationInfo {
  code: string;
  tx_hash: string;
}

export interface TokenPrice {
  asset: string;
  price: string;
  ts: string;
}

// total: total fund raised in USD
// tokens: number tokens claimable by donors
export interface DonationStats {
  total: string;
  tokens: string;
}

export type CampaignStatus = "open" | "paused" | "closed";

interface Donation {
  amount: string;
  usd_amount: string;
  asset: string;
  token: string;
  price: string;
  tx_hash: string;
  status: UserDonationStatus;
  ts: string;
}

export interface UserStats {
  total: string;
  tokens: string;
  staked: string;
  reward: string;
  status: UserRewardStatus;
  ts: string;
}

export type UserDonationStatus = "unconfirmed" | "confirmed" | "failed";

export type UserRewardStatus = "none" | "staking" | "unstaking";
