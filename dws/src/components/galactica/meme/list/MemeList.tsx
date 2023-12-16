"use client";

import { useLatestMeme } from "@/hooks/galactica/meme/useMeme";
import MemeListItem from "./MemeListItem";
import {
  ReactElement,
  memo,
  useCallback,
  useEffect,
  useRef,
  useState,
  useTransition,
} from "react";
import { roboto } from "@/app/fonts";
import { MemeUpload } from "@/api/galactica/meme/meme.type";

interface MemeListProps {
  memes: MemeUpload[];
  loadMore: () => void;
}

const MemeList = ({
  memes,
  loadMore,
}: MemeListProps): ReactElement<MemeListProps> => {
  const observerTarget = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting) {
          loadMore();
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
  }, [observerTarget, loadMore]);

  if (memes.length === 0) {
    return (
      <div className="py-8 flex items-center justify-center">
        No memes for today
      </div>
    );
  }

  return (
    <div className={roboto.className + " relative"}>
      {memes.map((meme) => (
        <MemeListItem {...meme} isServerMeme key={meme.fileId} />
      ))}

      <div ref={observerTarget}></div>
    </div>
  );
};

export default memo(MemeList);
