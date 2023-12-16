import { getLatestMeme, getMemeImage } from "@/api/galactica/meme/meme";
import { MemeUpload } from "@/api/galactica/meme/meme.type";
import { PaginationRequest } from "@/utils";
import { useCallback, useMemo, useState } from "react";

const LIMIT = 5;

export const useLatestMeme = () => {
  const [memes, setMemes] = useState<MemeUpload[]>([]);
  const [page, setPage] = useState<number>(0);
  const [total, setTotal] = useState<number>(0);

  const [loading, setLoading] = useState(false);

  const hasNext = useMemo(() => {
    return page * LIMIT < total;
  }, [page, total]);

  const loadInit = useCallback(async () => {
    const paginationSettings: PaginationRequest = {
      offset: 0,
      limit: LIMIT,
    };
    const { data, pagination } = await getLatestMeme(paginationSettings);
    if (data.length === 0) {
      // Avoid state mutation on empty data
      return;
    }

    setMemes(data);
    setPage(1);
    setTotal(pagination.total);
  }, []);

  const loadMore = async () => {
    if (loading) {
      return;
    }

    setLoading(true);

    const paginationSettings: PaginationRequest = {
      offset: page * LIMIT,
      limit: LIMIT,
    };
    const { data } = await getLatestMeme(paginationSettings);
    if (data.length === 0) {
      // Avoid state mutation on empty data
      return;
    }

    setMemes([...memes, ...data]);
    setPage((page) => page + 1);
    setLoading(false);
  };

  return { memes, hasNext, loading, loadInit, loadMore };
};

export const useMemeImage = () => {
  const [loading, setLoading] = useState(false);
  const [url, setUrl] = useState("");

  const getImage = useCallback(async (id: string) => {
    setLoading(true);

    const url = await getMemeImage(id);
    if (url == null) {
      setLoading(false);
      return;
    }

    setUrl(url);
    setLoading(false);
  }, []);

  return {
    loading,
    url,
    getMemeImage: getImage,
  };
};
