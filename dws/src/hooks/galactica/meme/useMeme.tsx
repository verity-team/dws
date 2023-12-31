import {
  getLatestMeme,
  getMemeImage,
  getPreviewMeme,
} from "@/api/galactica/meme/meme";
import { MemeUpload } from "@/api/galactica/meme/meme.type";
import { MemeFilter } from "@/components/galactica/meme/meme.type";
import { PaginationRequest } from "@/utils";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

const LIMIT = 5;

interface UseLatestMemeConfig {
  requireInit?: boolean;
}

export const useLatestMeme = (
  filter?: MemeFilter,
  config?: UseLatestMemeConfig
) => {
  const [memes, setMemes] = useState<MemeUpload[]>([]);
  const [page, setPage] = useState<number>(0);
  const [total, setTotal] = useState<number>(0);

  const loadingState = useRef(false);

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
    if (!config?.requireInit) {
      return;
    }

    const init = async () => {
      await loadInit();
      loadingState.current = false;
    };

    init();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filter?.status]);

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

  const removeMeme = useCallback((memeId: string) => {
    setMemes((memes) => memes.filter((meme) => meme.fileId !== memeId));
  }, []);

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
    removeMeme,
  };
};

export const usePreviewMeme = () => {
  const [memes, setMemes] = useState<MemeUpload[]>([]);

  const loadInit = useCallback(async () => {
    const { data } = await getPreviewMeme();
    if (data.length === 0) {
      // Avoid state mutation on empty data
      return;
    }

    setMemes(data);
  }, []);

  useEffect(() => {
    loadInit();
  }, []);

  return { memes };
};

export const useMemeImage = () => {
  const [url, setUrl] = useState("");

  const loadingState = useRef(true);

  const getImage = useCallback(async (id: string) => {
    loadingState.current = true;

    try {
      const blob = await getMemeImage(id);
      if (blob == null) {
        return;
      }

      setUrl(URL.createObjectURL(blob));
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
