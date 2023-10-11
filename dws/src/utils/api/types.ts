export interface FailedResponse {
  code: string;
  message: string;
}

export interface DonationData {
  prices: TokenInfo[];
  stats: DonationStats;
  receiving_address: string;
  status: "open" | "paused" | "closed";
}

export interface UserDonationData {
  donations: Donation[];
  stats: UserStats;
}

export interface AffiliateDonationInfo {
  code: string;
  tx_hash: string;
}

interface TokenInfo {
  asset: string;
  price: string;
  ts: string;
}

interface DonationStats {
  total: string;
  tokens: string;
}

interface Donation {
  amount: string;
  usd_amount: string;
  asset: string;
  token: string;
  price: string;
  tx_hash: string;
  status: "unconfirmed" | "confirmed" | "failed";
  ts: string;
}

interface UserStats {
  total: string;
  tokens: string;
  staked: string;
  reward: string;
  status: "none" | "staking" | "unstaking";
  ts: string;
}
