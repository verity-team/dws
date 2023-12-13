"use client";

import { useLatestMeme } from "@/hooks/galactica/meme/useMeme";
import { useEffect } from "react";

const MemeList = () => {
  const { memes, loadInit } = useLatestMeme();

  useEffect(() => {
    loadInit();
  }, []);

  if (memes.length === 0) {
    return <div>No memes for today</div>;
  }

  console.log(memes);

  return (
    <div>
      {memes.map((meme) => (
        <div key={meme.fileId}>{meme.caption}</div>
      ))}
    </div>
  );
};

export default MemeList;
