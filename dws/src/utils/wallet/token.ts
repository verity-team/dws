import BN from "bn.js";

export interface TokenInfo {
  symbol: string;
  contractAddress: string;
  decimals: number;
}

export type AvailableToken = "ETH" | "USDT" | "LINK";
export type AvailableWallet = "MetaMask" | "WalletConnect";

export const availableTokens: Array<AvailableToken> = ["ETH", "USDT"];
export const availableWallets: Array<AvailableWallet> = [
  "MetaMask",
  "WalletConnect",
];

export const contractAddrMap = new Map<AvailableToken, TokenInfo>([
  [
    "LINK",
    {
      symbol: "LINK",
      contractAddress: "0x779877A7B0D9E8603169DdbD7836e478b4624789",
      decimals: 18,
    },
  ],
  [
    "USDT",
    {
      symbol: "USDT",
      contractAddress: "0xdAC17F958D2ee523a2206206994597C13D831ec7",
      decimals: 18,
    },
  ],
]);

// TODO: Do NOT forget to update this when launch
export const stableCoinPrice: Record<string, number> = {
  USDT: 1,
  USDC: 1,
  LINK: 1,
};

export const multipleOrderOf10 = (base: BN, decimals: number): BN => {
  const addedDecimals = new BN(10).pow(new BN(decimals));
  return base.mul(addedDecimals);
};
