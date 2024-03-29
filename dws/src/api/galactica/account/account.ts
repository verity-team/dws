import { Maybe } from "@/utils";
import {
  NonceInfo,
  VerifySignaturePayload,
  VerifySignatureResponse,
} from "./account.type";
import {
  baseGalacticaRequest,
  getDefaultJsonHeaders,
  safeFetch,
  safeParseJson,
} from "@/utils/baseApiV2";

export const requestNonce = async (): Promise<Maybe<NonceInfo>> => {
  const path = "/auth/nonce";
  const response = await safeFetch(() => baseGalacticaRequest("GET", { path }));
  if (response == null || !response.ok) {
    return null;
  }

  const data = await safeParseJson<NonceInfo>(response);
  return data;
};

export const verifyAccessToken = async (
  address: string,
  jwt: string
): Promise<boolean> => {
  const path = "/auth/verify/user";
  const payload = { address };

  const headers = getDefaultJsonHeaders(jwt);

  const response = await safeFetch(() =>
    baseGalacticaRequest("POST", { path, payload, headers, json: true })
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
  const path = "/auth/verify/siwe";
  const response = await safeFetch(() =>
    baseGalacticaRequest("POST", { path, payload, json: true })
  );
  if (response == null || !response.ok) {
    return null;
  }

  const result = await safeParseJson<VerifySignatureResponse>(response);
  return result;
};
