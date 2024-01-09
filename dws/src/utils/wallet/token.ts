import BN from "bn.js";
import tokens from "./token.json";

export interface TokenInfo {
  symbol: string;
  contractAddress: string;
  decimals: number;
}

export type AvailableToken = "ETH" | keyof typeof tokens;
export const availableTokens: Array<AvailableToken> = [
  ...(Object.keys(tokens) as AvailableToken[]),
  "ETH",
];

export type AvailableWallet = "MetaMask" | "WalletConnect";
export const availableWallets: Array<AvailableWallet> = [
  "MetaMask",
  "WalletConnect",
];

export const stableCoinPrice = Object.values(tokens)
  .filter((token) => token.stable)
  .reduce((prev, token) => ({ ...prev, [token.symbol]: 1 }));

export const multipleOrderOf10 = (base: BN, decimals: number): BN => {
  const addedDecimals = new BN(10).pow(new BN(decimals));
  return base.mul(addedDecimals);
};
