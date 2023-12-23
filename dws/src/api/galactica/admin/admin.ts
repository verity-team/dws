import { baseGalacticaRequest, getDefaultJsonHeaders } from "@/utils/baseApiV2";
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

  try {
    const response = await baseGalacticaRequest("PATCH", {
      path,
      payload,
      headers,
      json: true,
    });
    if (!response.ok) {
      throw new Error(`Bad response code ${response.status}`);
    }
  } catch (error) {
    console.error(
      "Failed to update meme status",
      JSON.stringify(error, null, 2)
    );
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

  try {
    const response = await baseGalacticaRequest("POST", {
      path,
      headers,
    });
    if (!response.ok) {
      throw new Error(`Bad response code ${response.status}`);
    }
  } catch (error) {
    console.error(
      "Failed to verify admin access token",
      JSON.stringify(error, null, 2)
    );
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

  let response = null;
  try {
    response = await baseGalacticaRequest("POST", {
      path,
      payload,
      json: true,
    });
    if (response == null || !response.ok) {
      throw new Error(`Bad response code ${response.status}`);
    }
  } catch (error) {
    console.error(
      "Failed to verify admin access token",
      JSON.stringify(error, null, 2)
    );
    return null;
  }

  try {
    const data = await response.json();
    return data.accessToken;
  } catch (error) {
    console.error("Cannot parse SignInWithCredentialsRequest's response");
    return null;
  }
};
