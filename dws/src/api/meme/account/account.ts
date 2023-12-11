import { Maybe } from "@/utils";
import { clientBaseRequest, HttpMethod } from "@/utils/baseAPI";
import {
  NonceInfo,
  VerifyAccessTokenPayload,
  VerifySignaturePayload,
  VerifySignatureResponse,
} from "./account.type";

export const requestNonce = async (): Promise<Maybe<NonceInfo>> => {
  const response = await clientBaseRequest("/api/auth/nonce", HttpMethod.GET);

  if (response == null || !response.ok) {
    return null;
  }

  try {
    const result = await response.json();
    return result;
  } catch {
    return null;
  }
};

export const verifyAccessToken = async (
  payload: VerifyAccessTokenPayload
): Promise<boolean> => {
  const response = await clientBaseRequest(
    "/api/auth/verify/jwt",
    HttpMethod.POST,
    payload
  );
  if (response == null || !response.ok) {
    return false;
  }

  // Expect 200 OK
  return true;
};

export const verifySignature = async (
  payload: VerifySignaturePayload
): Promise<Maybe<VerifySignatureResponse>> => {
  const response = await clientBaseRequest(
    "api/auth/verify/siwe",
    HttpMethod.POST,
    payload
  );
  if (response == null || !response.ok) {
    return null;
  }

  try {
    const result = await response.json();
    return result;
  } catch {
    return null;
  }
};
