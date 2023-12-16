"use client";

import { useLatestMeme } from "@/hooks/galactica/meme/useMeme";
import MemeListItem from "./MemeListItem";
import { useCallback, useEffect, useRef, useState, useTransition } from "react";
import { roboto } from "@/app/fonts";

const MemeList = () => {
  const {
    memes,
    loadInit,
    loadMore,
    hasNext,
    loading: memeLoading,
  } = useLatestMeme();

  // Show scroll to top sticky button
  const [showScrollToTop, setShowScrollToTop] = useState(false);
  const [isPending, startTransition] = useTransition();

  useEffect(() => {
    loadInit();
  }, []);

  const handleLoadMore = useCallback(async () => {
    if (!hasNext || memeLoading) {
      return;
    }

    await loadMore();
  }, [hasNext, memeLoading, loadMore]);

  const startLoadMoreTransition = useCallback(() => {
    if (isPending) {
      return;
    }

    if (
      window.innerHeight + document.documentElement.scrollTop <
      document.documentElement.offsetHeight * 0.9
    ) {
      return;
    }

    startTransition(handleLoadMore);
  }, [handleLoadMore, isPending]);

  const observerTarget = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting) {
          startLoadMoreTransition();
        }
      },
      { threshold: 1 }
    );

    const flag = observerTarget.current;
    if (flag == null) {
      return;
    }

    if (observerTarget.current) {
      observer.observe(flag);
    }

    return () => {
      observer.unobserve(flag);
    };
  }, [observerTarget, startLoadMoreTransition]);

  if (memes.length === 0) {
    return <div>No memes for today</div>;
  }

  return (
    <div className={roboto.className + " relative"}>
      {memes.map((meme) => (
        <MemeListItem meme={meme} key={meme.fileId} />
      ))}
      {memeLoading && <p>Loading...</p>}
      <div ref={observerTarget}></div>
    </div>
  );
};

export default MemeList;
