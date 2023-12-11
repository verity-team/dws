import { NonceInfo } from "@/api/meme/account/account.type";
import { SiweMessage } from "siwe";

export const createSiweMesage = (
  address: string,
  { nonce, expirationTime, issuedAt }: NonceInfo
): string => {
  const domain = window.location.host;
  const origin = window.location.origin;

  address = address.toLowerCase();

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
    address: "0x62e662Ffb36Ffb465378Bc8cAEd807a9181a1561",
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
