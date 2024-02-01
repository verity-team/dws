import {
  getLatestMeme,
  getMemeImage,
  getPreviewMeme,
  withFilter,
} from "@/api/galactica/meme/meme";
import { MemeUpload } from "@/api/galactica/meme/meme.type";
import { MemeFilter } from "@/components/galactica/meme/meme.type";
import { safeFetch } from "@/utils/baseApiV2";
import { useEffect, useMemo, useState } from "react";
import useSWRImmutable from "swr/immutable";

const LIMIT = 5;

const mergeMemeList = (source: MemeUpload[], incoming: MemeUpload[]) => {
  const memeIdSet = new Set();

  for (const meme of source) {
    memeIdSet.add(meme.fileId);
  }

  const result = [...source];
  for (const meme of incoming) {
    if (memeIdSet.has(meme.fileId)) {
      continue;
    }
    memeIdSet.add(meme.fileId);
    result.push(meme);
  }

  return result;
};

export const useLatestMeme = (filter?: MemeFilter) => {
  const [page, setPage] = useState(-1);
  const [ended, setEnded] = useState(false);
  const [loaded, setLoaded] = useState<number[]>([]);
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<MemeUpload[]>([]);
  const [dataTotal, setDataTotal] = useState(-1);

  const resetState = () => {
    setPage(0);
    setEnded(false);
    setLoaded([]);
    setLoading(false);
    setData([]);
    setDataTotal(-1);
  };

  useEffect(() => {
    resetState();
  }, [filter]);

  const loadInit = async () => {
    setLoading(true);

    const targetStatus = filter?.status ?? "APPROVED";
    const response = await getLatestMeme(
      { offset: 0, limit: LIMIT },
      { status: targetStatus }
    );
    if (response.data.length === 0 || response.data.length < LIMIT) {
      setEnded(true);
    }
    if (dataTotal === -1) {
      setDataTotal(response.pagination.total);
    }

    setData(response.data);

    setLoaded([0]);
    setPage(0);

    setLoading(false);
  };

  useEffect(() => {
    loadInit();
  }, [filter]);

  const loadMore = async () => {
    setLoading(true);

    const targetStatus = filter?.status ?? "APPROVED";
    const nextPage = page + 1;
    const offset = nextPage * LIMIT;

    const response = await getLatestMeme(
      { offset, limit: LIMIT },
      { status: targetStatus }
    );
    if (response.data.length === 0 || response.data.length < LIMIT) {
      setEnded(true);
    }

    if (dataTotal === -1) {
      setDataTotal(response.pagination.total);
    }

    setData(mergeMemeList(data, response.data));

    setLoaded((loadedPages) => [...loadedPages, nextPage]);
    setPage(nextPage);

    setLoading(false);
  };

  return {
    data,
    loading,
    ended,
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
