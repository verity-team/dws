"use client";

import SignInBtn from "../account/SignInBtn";
import MemeInput from "./input/MemeInput";
import MemeList from "./list/MemeList";
import { useLatestMeme } from "@/hooks/galactica/meme/useMeme";
import { useCallback, useState, ReactElement } from "react";
import { OptimisticMemeUpload } from "@/api/galactica/meme/meme.type";
import MemeListItem from "./list/MemeListItem";
import { MemeFilter } from "./meme.type";
import MemeNavbar from "./common/MemeNavbar";

interface MemeTimelineProps {
  filter?: MemeFilter;
}

// Default to load latest approved memes
const MemeTimeline = ({
  filter = {
    status: "APPROVED",
  },
}: MemeTimelineProps): ReactElement<MemeTimelineProps> => {
  const { memes, loadMore, isLoading, hasNext } = useLatestMeme(filter);
  const [userMemes, setUserMemes] = useState<OptimisticMemeUpload[]>([]);

  const handleLoadMore = async () => {
    if (
      isLoading ||
      !hasNext ||
      window.innerHeight + document.documentElement.scrollTop <
        document.documentElement.offsetHeight * 0.9
    ) {
      return;
    }

    await loadMore();
  };

  // Optimistically update the meme list UI after user upload their meme
  const handleMemeUpload = useCallback((meme: OptimisticMemeUpload) => {
    setUserMemes((userMemes) => {
      if (userMemes.length === 0) {
        return [meme];
      }

      return [...userMemes, meme];
    });
  }, []);

  return (
    <>
      {/* <div className="flex items-center justify-between w-full">
        <h1 className="text-4xl font-bold">#Truthmemes</h1>
        <SignInBtn />
      </div> */}
      <MemeNavbar />
      <div className="mt-4">
        <MemeInput onUpload={handleMemeUpload} />
      </div>
      <div className="px-2">
        {userMemes.map((meme, i) => (
          <MemeListItem {...meme} isServerMeme={false} key={i} />
        ))}
        <MemeList
          memes={memes}
          loadMore={handleLoadMore}
          isLoading={isLoading}
        />
        {isLoading && (
          <div className="py-4 flex items-center justify-center">
            Loading more...
          </div>
        )}
      </div>
    </>
  );
};

export default MemeTimeline;
