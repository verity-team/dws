import { SiweMessage } from "siwe";

export const createSiweMesage = (address: string): string => {
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
    address,
    statement,
    uri: origin,
    version: "1.0.0",
    chainId: targetNetwork,
  });

  return message.prepareMessage();
};
