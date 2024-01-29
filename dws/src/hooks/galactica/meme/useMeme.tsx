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

const getLatestMemeKey =
  (filter?: MemeFilter) =>
  (pageIndex: number, previousPageData?: PaginationResponse<MemeUpload>) => {
    // End of meme list
    if (previousPageData?.data && !previousPageData.data.length) {
      return null;
    }

    const searchParams = withFilter(
      {
        offset: String(pageIndex * LIMIT),
        limit: String(LIMIT),
      },
      filter
    );

    const path = `/meme/latest?${new URLSearchParams(searchParams).toString()}`;
    return path;
  };

export const useLatestMeme = (filter?: MemeFilter) => {
  const { data, size, setSize, isLoading, error } = useSWRInfinite(
    getLatestMemeKey(filter),
    async (path: string): Promise<MemeUpload[]> => {
      const response = await baseGalacticaRequest("GET", { path });

      if (response == null || !response.ok) {
        return [];
      }

      const data =
        await safeParseJson<PaginationResponse<MemeUpload>>(response);
      if (!data?.data) {
        return [];
      }

      return data.data;
    }
  );

  const [ended, setEnded] = useState(false);

  useEffect(() => {
    // Keep trying for page 1
    if (!data) {
      return;
    }

    const lastData = data.at(data.length - 1);
    if (!lastData) {
      return;
    }

    if (lastData.length === 0) {
      setEnded(true);
    }
  }, [data]);

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
    if (ended) {
      return;
    }
    setSize((size) => size + 1);
  };

  return {
    memes,
    size,
    isLoading,
    error,
    loadMore,
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
