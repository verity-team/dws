import { NonceInfo } from "@/api/galactica/account/account.type";
import { SiweMessage } from "siwe";
import { getAddress } from "viem";

export const createSiweMesage = (
  address: string,
  { nonce, expirationTime, issuedAt }: NonceInfo
): string => {
  const domain = window.location.host;
  const origin = window.location.origin;

  let targetNetwork = 1;
  try {
    targetNetwork = parseInt(
      process.env.NEXT_PUBLIC_TARGET_NETWORK_ID ?? "0x1",
      16
    );
  } catch {
    // Ignore, use default value
  }

  const statement = "Welcome to Truth Memes";
  const message = new SiweMessage({
    domain,
    address: getAddress(address),
    statement,
    uri: origin,
    version: "1",
    chainId: targetNetwork,
    nonce,
    issuedAt,
    expirationTime,
  });

  return message.prepareMessage();
};
