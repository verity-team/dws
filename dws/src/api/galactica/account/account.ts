import { Maybe } from "@/utils";
import { clientBaseRequest, HttpMethod } from "@/utils/baseAPI";
import {
  NonceInfo,
  VerifyAccessTokenPayload,
  VerifySignaturePayload,
  VerifySignatureResponse,
} from "./account.type";

export const requestNonce = async (): Promise<Maybe<NonceInfo>> => {
  const response = await clientBaseRequest(
    "/auth/nonce",
    HttpMethod.GET,
    null,
    process.env.NEXT_PUBLIC_GALACTICA_API_URL
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

export const verifyAccessToken = async (address: string): Promise<boolean> => {
  const response = await clientBaseRequest(
    "/auth/verify/user",
    HttpMethod.POST,
    { address },
    process.env.NEXT_PUBLIC_GALACTICA_API_URL,
    true
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
    "/auth/verify/siwe",
    HttpMethod.POST,
    payload,
    process.env.NEXT_PUBLIC_GALACTICA_API_URL
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
