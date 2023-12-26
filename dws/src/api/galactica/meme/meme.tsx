import {
  HttpMethod,
  clientBaseRequest,
  clientFormRequest,
} from "@/utils/baseAPI";
import { MemeUpload, MemeUploadDTO } from "./meme.type";
import { Maybe, PaginationRequest, PaginationResponse } from "@/utils";
import { MemeFilter } from "@/components/galactica/meme/meme.type";
import { baseGalacticaRequest } from "@/utils/baseApiV2";

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
  pagination: PaginationRequest,
  filter?: MemeFilter
): Promise<PaginationResponse<MemeUpload>> => {
  const searchParams = withFilter(
    {
      offset: pagination.offset.toString(),
      limit: pagination.limit.toString(),
    },
    filter
  );

  const response = await clientBaseRequest(
    "/meme/latest?" + new URLSearchParams(searchParams),
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

export const getSingleMeme = async (id: string): Promise<Maybe<MemeUpload>> => {
  const path = `/meme/${id}`;

  let response = null;
  try {
    response = await baseGalacticaRequest("GET", { path });
    if (response == null || !response.ok) {
      return null;
    }
  } catch (error) {
    console.error("Request error:", JSON.stringify(error, null, 2));
    return null;
  }

  try {
    const data = await response.json();
    return data;
  } catch {
    console.error("Failed to parse data");
    return null;
  }
};

export const getMemeImage = async (id: string): Promise<Maybe<string>> => {
  const path = `/meme/image/${id}`;

  let response = null;
  try {
    response = await baseGalacticaRequest("GET", { path });
    if (response == null || !response.ok) {
      return null;
    }
  } catch (error) {
    console.error("Request error:", JSON.stringify(error, null, 2));
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

const withFilter = (
  searchParams: Record<string, string>,
  filter?: MemeFilter
): Record<string, string> => {
  if (filter == null || filter.status == null) {
    return searchParams;
  }

  return { ...searchParams, status: filter.status };
};
