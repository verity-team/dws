import toast from "react-hot-toast";
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

export const requestAccounts = async (): Promise<Array<string>> => {
  const ethereum = window.ethereum;
  if (ethereum == null) {
    toast.error("No Ethereum wallet installed");
    return [];
  }

  // Request wallet (connected) accounts
  let accounts = [];
  try {
    accounts = await ethereum.request({
      method: "eth_requestAccounts",
    });
    if (accounts == null || !Array.isArray(accounts)) {
      return [];
    }
  } catch (err) {
    console.warn({ err });
    return [];
  }

  // Request mainnet to continue
  const isMainnet = await requestTargetNetwork();
  if (!isMainnet) {
    return [];
  }

  return accounts;
};

// Request mainnet
const requestTargetNetwork = async (): Promise<boolean> => {
  const ethereum = window.ethereum;
  if (ethereum == null) {
    toast.error("No Ethereum wallet installed");
    return false;
  }

  // Get config from .env, or else fallback to mainnet config
  const targetNetworkId = process.env.NEXT_PUBLIC_TARGET_NETWORK_ID ?? "0x1";
  const targetNetworkRPC =
    process.env.NEXT_PUBLIC_TARGET_NETWORK_RPC ?? "https://eth.llamarpc.com";

  try {
    await ethereum.request({
      method: "wallet_switchEthereumChain",
      params: [{ chainId: targetNetworkId }],
    });
    return true;
  } catch (error: any) {
    if (error.code === 4902) {
      const success = await requestAddEthChain(
        targetNetworkId,
        targetNetworkRPC
      );
      if (!success) {
        console.warn(error);
        return false;
      }
      return true;
    }

    console.warn(error);
    return false;
  }
};

const requestAddEthChain = async (
  chainId: string,
  rpcUrl: string
): Promise<boolean> => {
  const ethereum = window.ethereum;
  if (ethereum == null) {
    toast.error("No Ethereum wallet installed");
    return false;
  }

  try {
    await ethereum.request({
      method: "wallet_addEthereumChain",
      params: [{ chainId, rpcUrl }],
    });
  } catch (error: any) {
    console.warn(error.message);
    return false;
  }

  return true;
};

export const getWalletShorthand = (wallet: string): string => {
  return `${wallet.slice(0, 6)}...${wallet.slice(-4)}`;
};
