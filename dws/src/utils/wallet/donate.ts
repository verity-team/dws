import { toWei } from "web3-utils";
import {
  AvailableToken,
  AvailableWallet,
  TokenInfo,
  contractAddrMap,
} from "./token";
import { encodeFunctionCall } from "web3-eth-abi";
import {
  erc20ABI,
  getPublicClient,
  prepareSendTransaction,
  prepareWriteContract,
  sendTransaction,
  writeContract,
} from "@wagmi/core";
import { formatEther, formatUnits, getContract, parseEther } from "viem";
import { Maybe } from "@/utils";
import Decimal from "decimal.js";
import toast from "react-hot-toast";
import { NOT_ENOUGH_ERR, NO_BALANCE_ERR } from "../const";

const getContractData = (
  receiver: string,
  amount: number,
  decimals: number
): Maybe<string> => {
  const TRANSFER_FUNCTION_ABI = {
    constant: false,
    inputs: [
      { name: "_to", type: "address" },
      { name: "_value", type: "uint256" },
    ],
    name: "transfer",
    outputs: [],
    payable: false,
    stateMutability: "nonpayable",
    type: "function",
  };

  const contractAmount = new Decimal(amount).mul(new Decimal(10).pow(decimals));
  if (contractAmount.lessThanOrEqualTo(0)) {
    toast.error("The amount input is too small to make a transaction");
    return null;
  }

  return encodeFunctionCall(TRANSFER_FUNCTION_ABI, [
    receiver,
    contractAmount.toString(),
  ]);
};

export const donate = async (
  from: string,
  amount: number,
  token: AvailableToken,
  provider: AvailableWallet = "MetaMask"
): Promise<Maybe<string>> => {
  let txHash = null;

  const walletBalance = await getBalance(token, from);
  console.log(walletBalance);

  const balance = Number(walletBalance);
  if (walletBalance == null || isNaN(balance)) {
    return NO_BALANCE_ERR;
  }

  if (amount > balance) {
    return NOT_ENOUGH_ERR;
  }

  if (provider === "MetaMask") {
    if (token === "ETH") {
      txHash = await donateETH(from, amount);
    } else {
      txHash = await donateERC(from, token, amount);
    }
  }

  if (provider === "WalletConnect") {
    if (token === "ETH") {
      txHash = await donateETHWagmi(amount);
    } else {
      txHash = await donateERCWagmi(token, amount);
    }
  }

  return txHash;
};

export const donateETH = async (
  from: string,
  amount: number
): Promise<Maybe<string>> => {
  const ethereum = window.ethereum;
  if (ethereum == null) {
    console.warn("Window does not have ethereum object");
    return null;
  }

  const receiveWallet = process.env.NEXT_PUBLIC_DONATE_PUBKEY;
  if (receiveWallet == null) {
    console.warn("Receive wallet not set");
    return null;
  }

  return await ethereum.request({
    method: "eth_sendTransaction",
    params: [
      {
        from,
        to: receiveWallet,
        value: parseInt(toWei(amount, "ether")).toString(16),
        // gasLimit: '0x5028', // Customizable by the user during MetaMask confirmation.
        // maxPriorityFeePerGas: '0x3b9aca00', // Customizable by the user during MetaMask confirmation.
        // maxFeePerGas: '0x2540be400', // Customizable by the user during MetaMask confirmation.
      },
    ],
  });
};

export const donateETHWagmi = async (
  amount: number
): Promise<Maybe<string>> => {
  const receiveWallet = process.env.NEXT_PUBLIC_DONATE_PUBKEY;
  if (receiveWallet == null) {
    console.warn("Receive wallet not set");
    return null;
  }

  const txConfig = await prepareSendTransaction({
    to: receiveWallet,
    value: parseEther(amount.toString()),
  });

  const { hash } = await sendTransaction(txConfig);
  return hash;
};

export const donateERC = async (
  from: string,
  token: AvailableToken,
  amount: number
): Promise<Maybe<string>> => {
  const ethereum = window.ethereum;
  if (ethereum == null) {
    return null;
  }

  const receiveWallet = process.env.NEXT_PUBLIC_DONATE_PUBKEY;
  if (receiveWallet == null) {
    return null;
  }

  const tokenInfo = contractAddrMap.get(token);
  if (tokenInfo == null) {
    console.warn("Unknown ERC-20 contract address");
    return null;
  }

  const data = getContractData(receiveWallet, amount, tokenInfo.decimals);
  if (data == null) {
    return;
  }

  return await ethereum.request({
    method: "eth_sendTransaction",
    params: [
      {
        from,
        to: tokenInfo.contractAddress,
        data,
      },
    ],
  });
};

export const donateERCWagmi = async (
  token: AvailableToken,
  amount: number
): Promise<Maybe<string>> => {
  const receiveWallet = process.env.NEXT_PUBLIC_DONATE_PUBKEY;
  if (receiveWallet == null) {
    return null;
  }

  const tokenInfo = contractAddrMap.get(token);
  if (tokenInfo == null) {
    console.warn("Unknown ERC-20 contract address");
    return null;
  }

  const { request } = await prepareWriteContract({
    address: tokenInfo.contractAddress as any,
    abi: erc20ABI,
    functionName: "transfer",
    args: [receiveWallet as any, parseEther(amount.toString())],
  });

  const { hash } = await writeContract(request);
  return hash;
};

export const getBalance = async (
  token: AvailableToken,
  walletAddress: string
) => {
  const client = getPublicClient();

  if (token === "ETH") {
    try {
      const balance = await client.getBalance({
        address: walletAddress as `0x${string}`,
      });
      return formatEther(balance);
    } catch (error) {
      console.error("Cannot get wallet ETH balance");
      return null;
    }
  }

  const tokenInfo = contractAddrMap.get(token);
  if (tokenInfo == null) {
    console.warn("Unknown ERC-20 contract address");
    return null;
  }

  try {
    const contract = getContract({
      address: tokenInfo.contractAddress as `0x${string}`,
      abi: erc20ABI,
      publicClient: client,
    });

    const result = await contract.read.balanceOf([
      walletAddress as `0x${string}`,
    ]);
    console.log(result);

    return formatUnits(result, tokenInfo.decimals);
  } catch (error) {
    console.error("Cannot read ERC-20 balance", error);
    return null;
  }
};

// TODO: Check number boundary. Use BN if needed
export const exchangeToReward = (
  amount: number,
  tokenPrice: number,
  rewardTokenPrice: number
): number => {
  if (isNaN(tokenPrice)) {
    throw new Error("Source token price is NaN");
  }

  if (isNaN(rewardTokenPrice)) {
    throw new Error("Reward token price is Nan");
  }

  const reward = Math.ceil((amount * tokenPrice) / rewardTokenPrice);
  if (isNaN(reward) || reward <= 0) {
    return 0;
  }
  return reward;
};
