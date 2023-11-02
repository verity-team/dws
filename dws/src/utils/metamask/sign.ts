import { Maybe } from "../types";

export const requestSignature = async (
  account: string,
  message: string
): Promise<Maybe<string>> => {
  const ethereum = window.ethereum;
  if (ethereum == null) {
    console.warn("Window does not have ethereum object");
    return null;
  }

  const encodedMessage = `0x${Buffer.from(message, "utf-8").toString("hex")}`;

  const signature = await ethereum.request({
    method: "personal_sign",
    params: [encodedMessage, account],
  });

  // Ensure type safety because ethereum.request return unknown type
  if (typeof signature !== "string") {
    return null;
  }

  return signature;
};
