// Interface for /user/data/{address} response
export interface UserDonationData {
  donations: Donation[];
  user_data: UserStats;
}

// Interface for a single donation entry
export interface Donation {
  amount: string;
  usd_amount: string;
  asset: string;
  tokens: string;
  price: string;
  tx_hash: string;
  status: UserDonationStatus;
  ts: string;
}

// Interface for user stats
//
// The affiliate code is deliberately absent: /user/data/{address} is
// unauthenticated, so the referral code is only served over the signature
// protected /affiliate/code endpoint (see requestNewAffiliateCode).
export interface UserStats {
  total: string;
  tokens: string;
  staked: string;
  reward: string;
  status: UserRewardStatus;
  ts: string;
}

// Status of a transaction
export type UserDonationStatus = "unconfirmed" | "confirmed" | "failed";

// Status of user claimed reward
export type UserRewardStatus = "none" | "staking" | "unstaking";
