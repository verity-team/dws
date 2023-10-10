import BN from "bn.js";

export type AvailableToken = "ETH" | "USDT" | "LINK";

export interface TokenInfo {
  symbol: string;
  contractAddress: string;
  decimals: number;
}

export const multipleOrderOf10 = (base: BN, decimals: number): BN => {
  const addedDecimals = new BN(10).pow(new BN(decimals));
  return base.mul(addedDecimals);
};
