import {
  baseGalacticaRequest,
  getDefaultJsonHeaders,
  safeFetch,
  safeParseJson,
} from "@/utils/baseApiV2";
import {
  ChangeMemeStatusRequest,
  SignInWithCredentialsRequest,
} from "./admin.type";
import { MemeUploadStatus } from "@/components/galactica/meme/meme.type";
import { Maybe } from "@/utils";

const requestChangeMemeStatus = async (
  accessToken: string,
  memeId: string,
  status: MemeUploadStatus
): Promise<boolean> => {
  const path = `/meme/${memeId}/status`;
  const payload: ChangeMemeStatusRequest = {
    status,
  };

  const headers = getDefaultJsonHeaders();
  headers.append("Authorization", `Bearer ${accessToken}`);

  const response = await safeFetch(
    () => baseGalacticaRequest("PATCH", { path, payload, headers, json: true }),
    "Failed to update meme status"
  );
  if (response == null || !response.ok) {
    return false;
  }

  return true;
};

export const requestApproveMeme = async (
  accessToken: string,
  memeId: string
): Promise<boolean> => {
  return await requestChangeMemeStatus(accessToken, memeId, "APPROVED");
};

export const requestDeclineMeme = async (
  accessToken: string,
  memeId: string
): Promise<boolean> => {
  return await requestChangeMemeStatus(accessToken, memeId, "DENIED");
};

export const requestRevertMemeReview = async (
  accessToken: string,
  memeId: string
): Promise<boolean> => {
  return await requestChangeMemeStatus(accessToken, memeId, "PENDING");
};

export const requestAccessTokenVerification = async (
  accessToken: string
): Promise<boolean> => {
  const path = "/auth/verify/admin";

  const headers = getDefaultJsonHeaders();
  headers.append("Authorization", `Bearer ${accessToken}`);

  const response = await safeFetch(
    () => baseGalacticaRequest("POST", { path, headers }),
    "Failed to verify admin's access token"
  );
  if (response == null || !response.ok) {
    return false;
  }

  return true;
};

export const requestSignInWithCredentials = async (
  username: string,
  password: string
): Promise<Maybe<string>> => {
  const path = "/auth/signin";
  const payload: SignInWithCredentialsRequest = {
    username,
    password,
  };

  const response = await safeFetch(
    () => baseGalacticaRequest("POST", { path, payload, json: true }),
    "Failed to verify admin's access token"
  );
  if (response == null || !response.ok) {
    return null;
  }

  const data = await safeParseJson<{ accessToken: string }>(response);
  if (data == null) {
    return null;
  }

  return data.accessToken;
};
