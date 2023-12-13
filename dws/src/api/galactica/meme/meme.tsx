import {
  HttpMethod,
  clientBaseRequest,
  clientFormRequest,
} from "@/utils/baseAPI";
import { MemeUpload, MemeUploadDTO } from "./meme.type";
import { Maybe, PaginationRequest, PaginationResponse } from "@/utils";

export const uploadMeme = async ({
  meme,
  caption,
  userId,
}: MemeUploadDTO): Promise<boolean> => {
  const formData = new FormData();

  // Construct formData
  formData.append("lang", "en");
  formData.append("tags", "#meme");

  formData.append("userId", userId);
  formData.append("caption", caption);

  formData.append("fileName", meme);

  const response = await clientFormRequest(
    "/meme",
    formData,
    process.env.NEXT_PUBLIC_GALACTICA_API_URL,
    true
  );
  if (response == null || !response.ok) {
    return false;
  }

  return true;
};

export const getLatestMeme = async (
  pagination: PaginationRequest
): Promise<PaginationResponse<MemeUpload>> => {
  const response = await clientBaseRequest(
    "/meme/latest?" +
      new URLSearchParams({
        offset: pagination.offset.toString(),
        limit: pagination.limit.toString(),
      }),
    HttpMethod.GET,
    null,
    process.env.NEXT_PUBLIC_GALACTICA_API_URL,
    true
  );

  const defaultData = {
    data: [],
    pagination: {
      ...pagination,
      total: 0,
    },
  };

  if (response == null || !response.ok) {
    console.error("Failed to get latest memes");
    return defaultData;
  }

  try {
    const data = await response.json();
    return data;
  } catch {
    console.error("Failed to parse data");
    return defaultData;
  }
};

export const getMemeImage = async (id: string): Promise<Maybe<string>> => {
  const response = await clientBaseRequest(
    `/meme/image/${id}`,
    HttpMethod.GET,
    null,
    process.env.NEXT_PUBLIC_GALACTICA_API_URL,
    true
  );

  if (response == null || !response.ok) {
    return null;
  }

  try {
    const file = await response.blob();
    return URL.createObjectURL(file);
  } catch {
    console.error("Cannot parse response as blob");
    return null;
  }
};
