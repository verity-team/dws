"use client";

import SignInBtn from "../account/SignInBtn";
import MemeInput from "./input/MemeInput";
import MemeList from "./list/MemeList";
import { useLatestMeme, usePreviewMeme } from "@/hooks/galactica/meme/useMeme";
import {
  useCallback,
  useState,
  ReactElement,
  useContext,
  useEffect,
} from "react";
import { OptimisticMemeUpload } from "@/api/galactica/meme/meme.type";
import MemeListItem from "./list/MemeListItem";
import { MemeFilter } from "./meme.type";
import { ClientWallet } from "@/components/ClientRoot";

interface MemeTimelineProps {
  filter?: MemeFilter;
}

// Default to load latest approved memes
const MemeTimeline = ({
  filter = {
    status: "APPROVED",
  },
}: MemeTimelineProps): ReactElement<MemeTimelineProps> => {
  const account = useContext(ClientWallet);
  const { memes, loadMore, isLoading, hasNext, loadInit } = useLatestMeme(
    filter,
    { requireInit: false }
  );
  const { memes: previewMemes } = usePreviewMeme();

  const [userMemes, setUserMemes] = useState<OptimisticMemeUpload[]>([]);

  useEffect(() => {
    if (account == null || account === "") {
      return;
    }

    loadInit();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [account]);

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
      <div className="mt-4">
        <MemeInput onUpload={handleMemeUpload} />
      </div>
      <div className="px-2">
        {(account == null || account === "") && memes.length === 0 ? (
          <>
            <MemeList
              memes={previewMemes}
              loadMore={() => {}}
              isLoading={false}
            />
            <div className="fixed bottom-0 left-0 w-full bg-corange backdrop-blur-lg bg-opacity-50 mt-8 md:mt-12">
              <div className="w-full flex items-center justify-center px-4 p-8 space-x-1 text-xl cursor-pointer">
                <SignInBtn variant="text-only" />
                <span>to see more memes</span>
              </div>
            </div>
          </>
        ) : (
          <>
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
          </>
        )}
      </div>
    </>
  );
};

export default MemeTimeline;
