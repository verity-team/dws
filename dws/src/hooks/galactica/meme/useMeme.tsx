import { getLatestMeme, getMemeImage } from "@/api/galactica/meme/meme";
import { MemeUpload } from "@/api/galactica/meme/meme.type";
import { MemeFilter } from "@/components/galactica/meme/meme.type";
import { PaginationRequest } from "@/utils";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

const LIMIT = 5;

export const useLatestMeme = (filter?: MemeFilter) => {
  const [memes, setMemes] = useState<MemeUpload[]>([]);
  const [page, setPage] = useState<number>(0);
  const [total, setTotal] = useState<number>(0);

  const loadingState = useRef(true);

  const hasNext = useMemo(() => {
    if (page == 0) {
      return true;
    }

    return page * LIMIT < total;
  }, [page, total]);

  const loadInit = async () => {
    const paginationSettings: PaginationRequest = {
      offset: 0,
      limit: LIMIT,
    };
    const { data, pagination } = await getLatestMeme(
      paginationSettings,
      filter
    );
    if (data.length === 0) {
      // Avoid state mutation on empty data
      if (memes.length > 0) {
        setMemes([]);
      }
      return;
    }

    setMemes(data);
    setPage(1);
    setTotal(pagination.total);
  };

  useEffect(() => {
    loadInit().finally(() => {
      loadingState.current = false;
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filter]);

  const loadMore = async () => {
    if (loadingState.current || !hasNext) {
      return;
    }

    loadingState.current = true;
    try {
      const paginationSettings: PaginationRequest = {
        offset: page * LIMIT,
        limit: LIMIT,
      };
      const { data } = await getLatestMeme(paginationSettings, filter);
      if (data.length === 0) {
        // Avoid state mutation on empty data
        return;
      }
      setMemes((memes) => [...memes, ...data]);
      setPage((page) => page + 1);
    } catch (error) {
      console.error("Cannot load more", error);
    } finally {
      loadingState.current = false;
    }
  };

  const clear = () => {
    setMemes([]);
  };

  return {
    memes,
    hasNext,
    isLoading: loadingState.current,
    loadInit,
    loadMore,
    clear,
  };
};

export const useMemeImage = () => {
  const [url, setUrl] = useState("");

  const loadingState = useRef(true);

  const getImage = useCallback(async (id: string) => {
    loadingState.current = true;

    try {
      const url = await getMemeImage(id);
      if (url == null) {
        return;
      }

      setUrl(url);
    } catch (error) {
      console.error(error);
    } finally {
      loadingState.current = false;
    }
  }, []);

  return {
    loading: loadingState.current,
    url,
    getMemeImage: getImage,
  };
};
