import { toWei } from "web3-utils";
import {
  AvailableToken,
  AvailableWallet,
  contractAddrMap,
  multipleOrderOf10,
} from "./token";
import { encodeFunctionCall } from "web3-eth-abi";
import { BN } from "bn.js";
import {
  erc20ABI,
  prepareSendTransaction,
  prepareWriteContract,
  sendTransaction,
  writeContract,
} from "@wagmi/core";
import { parseEther } from "viem";
import { Maybe } from "@/utils";

const getContractData = (
  receiver: string,
  amount: number,
  decimals: number
): string => {
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

  const contractAmount = multipleOrderOf10(new BN(amount), decimals);
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

  return await ethereum.request({
    method: "eth_sendTransaction",
    params: [
      {
        from,
        to: tokenInfo.contractAddress,
        data: getContractData(receiveWallet, amount, tokenInfo.decimals),
        // gasLimit: '0x5028', // Customizable by the user during MetaMask confirmation.
        // maxPriorityFeePerGas: '0x3b9aca00', // Customizable by the user during MetaMask confirmation.
        // maxFeePerGas: '0x2540be400', // Customizable by the user during MetaMask confirmation.
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

// TODO: Check number boundary. Use BN if needed
export const exchangeToReward = (
  amount: number,
  tokenPrice: number
): number => {
  const rewardTokenPrice = Number(process.env.NEXT_PUBLIC_REWARD_PRICE);
  if (isNaN(tokenPrice)) {
    throw new Error("Reward price is NaN");
  }

  const reward = Math.ceil((amount * tokenPrice) / rewardTokenPrice);
  if (isNaN(reward) || reward <= 0) {
    return 0;
  }
  return reward;
};
