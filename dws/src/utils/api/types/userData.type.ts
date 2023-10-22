// Interface for /user/data/{address} response
export interface UserDonationData {
  donations: Donation[];
  stats: UserStats;
}

// Interface for a single donation entry
interface Donation {
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
export interface UserStats {
  total: string;
  tokens: string;
  staked: string;
  reward: string;
  status: UserRewardStatus;
  ts: string;
  affliate_code: string | "none";
}

// Status of a transaction
export type UserDonationStatus = "unconfirmed" | "confirmed" | "failed";

// Status of user claimed reward
export type UserRewardStatus = "none" | "staking" | "unstaking";
