"use client";

import { useLatestMeme } from "@/hooks/galactica/meme/useMeme";
import MemeListItem from "./MemeListItem";
import { useEffect } from "react";
import { roboto } from "@/app/fonts";

const MemeList = () => {
  const { memes, loadInit } = useLatestMeme();

  useEffect(() => {
    if (memes.length > 0) {
      return;
    }

    loadInit();
  }, []);

  if (memes.length === 0) {
    return <div>No memes for today</div>;
  }

  return (
    <div className={roboto.className}>
      {memes.map((meme) => (
        <div key={meme.fileId}>
          <MemeListItem meme={meme} />
        </div>
      ))}
    </div>
  );
};

export default MemeList;
