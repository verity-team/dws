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
  const isMainnet = await requestMainnet();
  if (!isMainnet) {
    return [];
  }

  return accounts;
};

// Request mainnet
const requestMainnet = async (): Promise<boolean> => {
  const ethereum = window.ethereum;
  if (ethereum == null) {
    toast.error("No Ethereum wallet installed");
    return false;
  }

  try {
    await ethereum.request({
      method: "wallet_switchEthereumChain",
      params: [{ chainId: "0x1" }],
    });
    return true;
  } catch (error: any) {
    if (error.code === 4902) {
      const success = await requestAddEthChain(
        "0x1",
        "https://eth.llamarpc.com"
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
