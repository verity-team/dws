import {
  getMemeImage,
  getPreviewMeme,
  withFilter,
} from "@/api/galactica/meme/meme";
import { MemeUpload } from "@/api/galactica/meme/meme.type";
import { MemeFilter } from "@/components/galactica/meme/meme.type";
import { PaginationResponse } from "@/utils";
import { baseGalacticaRequest, safeParseJson } from "@/utils/baseApiV2";
import { useEffect, useMemo, useState } from "react";
import useSWRImmutable from "swr/immutable";
import useSWRInfinite from "swr/infinite";

const LIMIT = 5;

const requestLatestMeme = async (
  queryParamters: string[]
): Promise<MemeUpload[]> => {
  const response = await baseGalacticaRequest("GET", {
    path: queryParamters[0],
  });

  if (response == null || !response.ok) {
    return [];
  }

  const data = await safeParseJson<PaginationResponse<MemeUpload>>(response);
  if (!data?.data) {
    return [];
  }

  return data.data;
};

export const useLatestMeme = (filter?: MemeFilter) => {
  const { data, size, setSize, isLoading, error } = useSWRInfinite(
    (pageIndex: number, previousPageData?: PaginationResponse<MemeUpload>) => {
      const targetStatus = filter?.status ?? "APPROVED";

      // End of meme list
      if (previousPageData?.data && !previousPageData.data.length) {
        return null;
      }

      if (pageIndex === 0) {
        return [
          `/meme/latest?offset=0&limit=5&status=${filter?.status}`,
          targetStatus,
        ];
      }

      const offset = pageIndex * LIMIT;
      const path = `/meme/latest?offset=${offset}&limit=${LIMIT}&status=${targetStatus}`;
      return [path, targetStatus];
    },
    requestLatestMeme,
    { revalidateFirstPage: false }
  );

  // Flat and dedupe the data to ready to use format
  const memes = useMemo(() => {
    if (!data) {
      return [];
    }

    const memeIdMap = new Map<string, boolean>();
    const result: MemeUpload[] = [];

    for (const page of data) {
      for (const meme of page) {
        // Skip duplicate memes
        if (memeIdMap.has(meme.fileId)) {
          continue;
        }

        memeIdMap.set(meme.fileId, true);
        result.push(meme);
      }
    }

    return result;
  }, [data]);

  // Load next page
  const loadMore = () => {
    setSize((size) => size + 1);
  };

  return {
    memes,
    size,
    isLoading,
    error,
    loadMore,
    setSize,
  };
};

export const usePreviewMeme = () => {
  const { data, isLoading } = useSWRImmutable("preview", getPreviewMeme);
  return { memes: data, isLoading };
};

export const useMemeImage = (fileId: string) => {
  const { data, isLoading } = useSWRImmutable(fileId, getMemeImage);

  const imageUrl = useMemo(() => {
    if (!data) {
      return null;
    }

    return URL.createObjectURL(data);
  }, [data]);

  return {
    imageRaw: data,
    imageUrl,
    isLoading,
  };
};
