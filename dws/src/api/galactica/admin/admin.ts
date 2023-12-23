import { baseGalacticaRequest } from "@/utils/baseApiV2";
import { ChangeMemeStatusRequest } from "./admin.type";
import { MemeUploadStatus } from "@/components/galactica/meme/meme.type";

const requestChangeMemeStatus = async (
  memeId: string,
  status: MemeUploadStatus
): Promise<boolean> => {
  const path = `/meme/${memeId}/status`;
  const payload: ChangeMemeStatusRequest = {
    status,
  };

  try {
    const response = await baseGalacticaRequest("PATCH", {
      path,
      payload,
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

export const approveMeme = async (memeId: string): Promise<boolean> => {
  return await requestChangeMemeStatus(memeId, "APPROVED");
};

export const declineMeme = async (memeId: string): Promise<boolean> => {
  return await requestChangeMemeStatus(memeId, "DENIED");
};

export const revertMemeReview = async (memeId: string): Promise<boolean> => {
  return await requestChangeMemeStatus(memeId, "PENDING");
};
