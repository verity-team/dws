"use client";

import MemeListItem from "./MemeListItem";
import { ReactElement, memo, useEffect, useRef } from "react";
import { MemeUpload } from "@/api/galactica/meme/meme.type";
import AdminMemeListItem from "@/components/admin/list/AdminMemeListItem";
import { roboto } from "@/app/fonts";

interface MemeListProps {
  memes: MemeUpload[];
  isLoading: boolean;
  admin?: boolean;
  loadMore: () => void;
  removeMeme?: (memeId: string) => void;
}

const MemeList = ({
  memes,
  admin = false,
  isLoading,
  loadMore,
  removeMeme = () => {},
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

    observer.observe(flag);
    return () => {
      observer.unobserve(flag);
    };
  }, [loadMore]);

  if (memes?.length === 0 && !isLoading) {
    return (
      <div className="py-8 flex items-center justify-center">
        No memes for today
      </div>
    );
  }

  return (
    <div className={roboto.className + " relative"}>
      {memes.map((meme) =>
        admin ? (
          <AdminMemeListItem
            {...meme}
            isServerMeme
            key={meme.fileId}
            removeMeme={removeMeme}
          />
        ) : (
          <MemeListItem {...meme} isServerMeme key={meme.fileId} />
        )
      )}
      <div ref={observerTarget}></div>
    </div>
  );
};

export default memo(MemeList);
