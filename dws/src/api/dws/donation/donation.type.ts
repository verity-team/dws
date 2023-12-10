// Interface for /donation/data response
export interface DonationData {
  prices: TokenPrice[];
  stats: DonationStats;
  receiving_address: string;
  status: CampaignStatus;
}

// Interface for a token price entry
export interface TokenPrice {
  asset: string;
  price: string;
  ts: string;
}

// Interface for current campaign stats
// total: total fund raised in USD
// tokens: number tokens claimable by donors
export interface DonationStats {
  total: string;
  tokens: string;
}

// Interface for campaign current result
// Donation activities should be postponed until the status is "open" again
export type CampaignStatus = "open" | "paused" | "closed";
