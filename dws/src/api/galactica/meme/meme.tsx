import { clientFormRequest } from "@/utils/baseAPI";
import { MemeUpload, MemeUploadDTO } from "./meme.type";
import { Maybe, PaginationRequest, PaginationResponse } from "@/utils";
import { MemeFilter } from "@/components/galactica/meme/meme.type";
import {
  baseGalacticaRequest,
  safeFetch,
  safeParseJson,
} from "@/utils/baseApiV2";
import toast from "react-hot-toast";

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

  try {
    const response = await clientFormRequest(
      "/meme",
      formData,
      process.env.NEXT_PUBLIC_GALACTICA_API_URL,
      true
    );
    if (response == null) {
      return false;
    }

    if (!response.ok) {
      if (response.status === 409) {
        toast("This meme have been uploaded before!", { icon: "👎" });
        return false;
      }

      toast.error(
        "Something wrong happend when uploading your meme. Please try again later"
      );
      return false;
    }
  } catch (error) {
    console.error("Cannot upload meme", error);
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

  const defaultData = {
    data: [],
    pagination: {
      ...pagination,
      total: -1,
    },
  };

  const path = `/meme/latest?${new URLSearchParams(searchParams).toString()}`;
  const response = await safeFetch(() => baseGalacticaRequest("GET", { path }));

  if (response == null || !response.ok) {
    if (response?.status === 429) {
      toast.error("Too many request. Please try again later");
    }
    console.error("Failed to get latest memes");
    return defaultData;
  }

  const data = await safeParseJson<PaginationResponse<MemeUpload>>(response);
  if (data == null) {
    return defaultData;
  }

  return data;
};

export const getPreviewMeme = async (): Promise<
  PaginationResponse<MemeUpload>
> => {
  const path = "/meme/preview";
  const response = await safeFetch(() => baseGalacticaRequest("GET", { path }));

  const defaultData: PaginationResponse<MemeUpload> = {
    data: [],
    pagination: {
      limit: 10,
      offset: 0,
      total: 0,
    },
  };

  if (response == null || !response.ok) {
    if (response?.status === 429) {
      toast.error("Too many request. Please try again later");
    }
    console.error("Failed to get latest memes");
    return defaultData;
  }

  const data = await safeParseJson<PaginationResponse<MemeUpload>>(response);
  if (!data) {
    return defaultData;
  }

  return data;
};

export const getSingleMeme = async (id: string): Promise<Maybe<MemeUpload>> => {
  const path = `/meme/${id}`;

  const response = await safeFetch(() => baseGalacticaRequest("GET", { path }));
  if (response == null) {
    return null;
  }

  if (!response.ok) {
    if (response.status === 429) {
      toast.error("Too many request. Please try again later");
    }
    return null;
  }

  const data = await safeParseJson<MemeUpload>(response);
  return data;
};

export const getMemeImage = async (id: string): Promise<Maybe<Blob>> => {
  const path = `/meme/image/${id}`;

  const response = await safeFetch(() => baseGalacticaRequest("GET", { path }));
  if (response == null) {
    return null;
  }

  if (!response.ok) {
    if (response.status === 429) {
      toast.error("Too many request");
    }
    return null;
  }

  try {
    const file = await response.blob();
    return file;
  } catch {
    console.error("Cannot parse response as blob");
    return null;
  }
};

export const withFilter = (
  searchParams: Record<string, string>,
  filter?: MemeFilter
): Record<string, string> => {
  if (filter == null || filter.status == null) {
    return searchParams;
  }

  return { ...searchParams, status: filter.status };
};
