import { verifyAccessToken } from "@/api/galactica/account/account";
import { Maybe } from "@/utils";
import { DWS_AT_KEY } from "@/utils/const";
import { JwtPayload, decode } from "jsonwebtoken";

export const getAccessToken = (): Maybe<string> => {
  const accessToken = localStorage.getItem(DWS_AT_KEY);
  if (accessToken == null || accessToken === "") {
    return null;
  }

  return accessToken;
};

export const removeAccessToken = (): void => {
  localStorage.removeItem(DWS_AT_KEY);
};

export const getAccessTokenPayload = (): Maybe<JwtPayload> => {
  const accessToken = getAccessToken();
  if (accessToken == null) {
    return null;
  }

  const payload = decode(accessToken, { json: true });
  if (payload == null) {
    return null;
  }

  return payload;
};

/**
 * This function will try to use the local storage access token to retrieve a wallet address in case there are none provided
 * @param walletAddress Ethereum wallet address - optional
 * @returns The wallet address used to verify the access token
 */
export const verifyToken = async (
  walletAddress?: string
): Promise<Maybe<string>> => {
  const accessToken = getAccessToken();
  if (accessToken == null) {
    return null;
  }

  let address = walletAddress;
  if (address == null) {
    address = getAccessTokenPayload()?.["address"];
    if (address == null) {
      removeAccessToken();
      return null;
    }
  }

  // Validate token using server
  const isValid = await verifyAccessToken(address);
  if (!isValid) {
    removeAccessToken();
    return null;
  }

  return address;
};
