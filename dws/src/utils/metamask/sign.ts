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

  const signature = await ethereum.request({
    method: "personal_sign",
    params: [message, account],
  });
  return signature;
};
