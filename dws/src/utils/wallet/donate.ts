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
import {
  Address,
  formatEther,
  formatUnits,
  getAddress,
  getContract,
  isAddress,
  isHex,
  parseUnits,
  toHex,
} from "viem";
import { Maybe } from "@/utils";
import Decimal from "decimal.js";
import toast from "react-hot-toast";
import { NOT_ENOUGH_ERR, NO_BALANCE_ERR } from "../const";

/**
 * Convert a donation amount to an exact integer amount of the token's base
 * units (wei for ETH). parseUnits silently rounds extra decimals, so an amount
 * more precise than the token allows is rejected instead.
 *
 * @param amount [number] the amount in whole tokens
 * @param decimals [number] the token decimals
 * @returns the base-unit amount, or null (with a toast) if it is not usable
 */
const toBaseUnits = (amount: number, decimals: number): Maybe<bigint> => {
  if (!Number.isFinite(amount)) {
    toast.error("Invalid amount");
    return null;
  }

  // Decimal prints plain notation; Number.toString may use an exponent
  const value = new Decimal(amount);
  if (value.decimalPlaces() > decimals) {
    toast.error(`The amount can have at most ${decimals} decimal places`);
    return null;
  }

  const units = parseUnits(value.toFixed(), decimals);
  if (units <= BigInt(0)) {
    toast.error("The amount input is too small to make a transaction");
    return null;
  }

  return units;
};

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

  const contractAmount = toBaseUnits(amount, decimals);
  if (contractAmount == null) {
    return null;
  }

  return encodeFunctionCall(TRANSFER_FUNCTION_ABI, [
    receiver,
    contractAmount.toString(),
  ]);
};

/**
 * The chain id donations must be sent on, from NEXT_PUBLIC_TARGET_NETWORK_ID
 * (hex, e.g. "0x1"); mainnet when unset, like the wagmi chain config.
 *
 * @returns the chain id, or null if the configured value is invalid
 */
export const getTargetChainId = (): Maybe<number> => {
  const raw = process.env.NEXT_PUBLIC_TARGET_NETWORK_ID ?? "0x1";
  if (!/^0x[0-9a-fA-F]+$/.test(raw)) {
    return null;
  }

  const chainId = parseInt(raw, 16);
  return Number.isSafeInteger(chainId) && chainId > 0 ? chainId : null;
};

/**
 * Return the donation address, but only if the build-time
 * NEXT_PUBLIC_DONATE_PUBKEY and the `receiving_address` served by the API are
 * both valid and the same address. The backend credits only payments to the
 * address it serves, so a stale or mistyped build must not be paid.
 *
 * @param apiAddress [string] `receiving_address` from /donation/data
 * @returns the checksummed address, or null if they are invalid or differ
 */
export const getVerifiedReceiver = (
  apiAddress: Maybe<string>
): Maybe<Address> => {
  const configured = process.env.NEXT_PUBLIC_DONATE_PUBKEY;
  if (
    configured == null ||
    apiAddress == null ||
    !isAddress(configured) ||
    !isAddress(apiAddress)
  ) {
    return null;
  }

  const receiver = getAddress(configured);
  if (receiver !== getAddress(apiAddress)) {
    console.warn("Donation address does not match the server address");
    return null;
  }

  return receiver;
};

export const donate = async (
  from: string,
  amount: number,
  token: AvailableToken,
  receiver: Address,
  provider: AvailableWallet = "MetaMask"
): Promise<Maybe<string>> => {
  let txHash = null;

  if (!isAddress(receiver)) {
    console.warn("Invalid donation address");
    return null;
  }

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
      txHash = await donateETH(from, receiver, amount);
    } else {
      txHash = await donateERC(from, receiver, token, amount);
    }
  }

  if (provider === "WalletConnect") {
    if (token === "ETH") {
      txHash = await donateETHWagmi(receiver, amount);
    } else {
      txHash = await donateERCWagmi(receiver, token, amount);
    }
  }

  return txHash;
};

export const donateETH = async (
  from: string,
  receiver: Address,
  amount: number
): Promise<Maybe<string>> => {
  const ethereum = window.ethereum;
  if (ethereum == null) {
    console.warn("Window does not have ethereum object");
    return null;
  }

  const value = toBaseUnits(amount, 18);
  if (value == null) {
    return null;
  }

  return await ethereum.request({
    method: "eth_sendTransaction",
    params: [
      {
        from,
        to: receiver,
        value: toHex(value),
        // gasLimit: '0x5028', // Customizable by the user during MetaMask confirmation.
        // maxPriorityFeePerGas: '0x3b9aca00', // Customizable by the user during MetaMask confirmation.
        // maxFeePerGas: '0x2540be400', // Customizable by the user during MetaMask confirmation.
      },
    ],
  });
};

export const donateETHWagmi = async (
  receiver: Address,
  amount: number
): Promise<Maybe<string>> => {
  const chainId = getTargetChainId();
  if (chainId == null) {
    console.warn("Invalid target network");
    return null;
  }

  const value = toBaseUnits(amount, 18);
  if (value == null) {
    return null;
  }

  // with a chainId wagmi refuses to send if the wallet is on another chain
  const txConfig = await prepareSendTransaction({
    chainId,
    to: receiver,
    value,
  });

  const { hash } = await sendTransaction(txConfig);
  return hash;
};

export const donateERC = async (
  from: string,
  receiver: Address,
  token: AvailableToken,
  amount: number
): Promise<Maybe<string>> => {
  const ethereum = window.ethereum;
  if (ethereum == null) {
    return null;
  }

  const tokenInfo = contractAddrMap.get(token);
  if (tokenInfo == null) {
    console.warn("Unknown ERC-20 contract address");
    return null;
  }

  const data = getContractData(receiver, amount, tokenInfo.decimals);
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
  receiver: Address,
  token: AvailableToken,
  amount: number
): Promise<Maybe<string>> => {
  const tokenInfo = contractAddrMap.get(token);
  if (tokenInfo == null) {
    console.warn("Unknown ERC-20 contract address");
    return null;
  }

  const chainId = getTargetChainId();
  if (chainId == null) {
    console.warn("Invalid target network");
    return null;
  }

  const units = toBaseUnits(amount, tokenInfo.decimals);
  if (units == null) {
    return null;
  }

  // with a chainId wagmi refuses to send if the wallet is on another chain
  const { request } = await prepareWriteContract({
    chainId,
    address: tokenInfo.contractAddress as any,
    abi: erc20ABI,
    functionName: "transfer",
    args: [receiver, units],
  });

  const { hash } = await writeContract(request);
  return hash;
};

// How long to wait for a donation to be mined before giving up on the receipt
const RECEIPT_TIMEOUT_MS = 15 * 60 * 1000;

// "replaced": the wallet cancelled the donation or sent another transaction
// with the same nonce; "unknown": no receipt (timeout or RPC failure)
export type DonationOutcome = "success" | "reverted" | "replaced" | "unknown";

/**
 * Wait for a donation transaction to be mined on the target chain.
 *
 * Uses the configured public client, so it works for both MetaMask and
 * WalletConnect. A speed-up (same transaction, higher fee) is followed to the
 * new hash; any other replacement is not a donation.
 *
 * @param hash [string] the donation transaction hash
 */
export const waitForDonation = async (
  hash: string
): Promise<DonationOutcome> => {
  const chainId = getTargetChainId();
  if (chainId == null || !isHex(hash) || hash.length !== 66) {
    console.warn("Cannot wait for donation receipt: invalid input");
    return "unknown";
  }

  let replaced = false;
  try {
    const receipt = await getPublicClient({
      chainId,
    }).waitForTransactionReceipt({
      hash,
      timeout: RECEIPT_TIMEOUT_MS,
      onReplaced: ({ reason }) => {
        replaced = reason !== "repriced";
      },
    });

    if (replaced) {
      return "replaced";
    }
    return receipt.status === "success" ? "success" : "reverted";
  } catch (error) {
    console.error("Cannot get donation receipt", error);
    return "unknown";
  }
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
  // console.log(
  //   "Amount",
  //   amount,
  //   "Price",
  //   tokenPrice,
  //   "Reward price",
  //   rewardTokenPrice
  // );

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
