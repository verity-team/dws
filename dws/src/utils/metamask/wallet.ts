import { Maybe, Undefinable } from "../types";
import { MetaMaskSDK } from "@metamask/sdk/dist/browser/es/src";

export const connectWallet = async (
  sdk: Undefinable<MetaMaskSDK>
): Promise<Maybe<string>> => {
  try {
    if (sdk == null) {
      return null;
    }
    const accounts = await sdk.connect();

    if (accounts == null || !Array.isArray(accounts)) {
      return null;
    }

    return accounts[0];
  } catch (err) {
    console.warn({ err });
    return null;
  }
};
