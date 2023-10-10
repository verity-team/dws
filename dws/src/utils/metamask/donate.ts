import { toWei } from "web3-utils";
import { Maybe, Nullable } from "../types";
import { AvailableToken } from "../token";
import { encodeFunctionCall } from "web3-eth-abi";

const contractAddrMap = new Map<AvailableToken, string>([
  ["LINK", "0x779877A7B0D9E8603169DdbD7836e478b4624789"],
]);

const getContractData = (receiver: string, amount: number): string => {
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
  return encodeFunctionCall(TRANSFER_FUNCTION_ABI, [receiver, amount]);
};

export const donate = async (
  from: string,
  amount: number,
  token: AvailableToken
): Promise<Nullable<string>> => {
  let txHash = null;

  if (token === "ETH") {
    txHash = await donateETH(from, amount);
  } else {
    txHash = await donateERC(from, token, amount);
  }

  if (txHash == null) {
    return null;
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

  try {
    return await ethereum.request<string>({
      method: "eth_sendTransaction",
      params: [
        {
          from,
          to: receiveWallet,
          value: toWei(amount, "ether"),
          // gasLimit: '0x5028', // Customizable by the user during MetaMask confirmation.
          // maxPriorityFeePerGas: '0x3b9aca00', // Customizable by the user during MetaMask confirmation.
          // maxFeePerGas: '0x2540be400', // Customizable by the user during MetaMask confirmation.
        },
      ],
    });
  } catch (err: any) {
    console.warn(err.message);
    return null;
  }
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

  const sendTo = contractAddrMap.get(token);
  if (sendTo == null) {
    console.warn("Unknown ERC-20 contract address");
    return null;
  }

  return await ethereum.request<string>({
    method: "eth_sendTransaction",
    params: [
      {
        from,
        to: sendTo,
        data: getContractData(receiveWallet, amount),
        // gasLimit: '0x5028', // Customizable by the user during MetaMask confirmation.
        // maxPriorityFeePerGas: '0x3b9aca00', // Customizable by the user during MetaMask confirmation.
        // maxFeePerGas: '0x2540be400', // Customizable by the user during MetaMask confirmation.
      },
    ],
  });
};

// TODO: Check number boundary. Use BN if needed
export const exchangeToReward = (
  amount: number,
  tokenPrice: number
): number => {
  const rewardTokenPrice = Number(process.env.NEXT_PUBLIC_REWARD_PRICE);
  if (rewardTokenPrice == null) {
    throw new Error("Reward price is NaN");
  }

  return Math.ceil((amount * tokenPrice) / rewardTokenPrice);
};
