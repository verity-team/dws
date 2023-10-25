import { Maybe } from "../types";

export const connectWallet = async (): Promise<Maybe<string>> => {
  const ethereum = window?.ethereum;
  if (ethereum == null) {
    return;
  }

  try {
    const accounts = await ethereum.request({ method: "eth_requestAccounts" });
    if (accounts == null || !Array.isArray(accounts)) {
      return null;
    }

    return accounts[0];
  } catch (err) {
    console.warn({ err });
    return null;
  }
};

export const getWalletShorthand = (wallet: string): string => {
  return `${wallet.slice(0, 6)}...${wallet.slice(-4)}`;
};
