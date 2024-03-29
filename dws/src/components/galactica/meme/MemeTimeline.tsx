"use client";

import MemeInput from "./input/MemeInput";
import MemeList from "./list/MemeList";
import { useLatestMeme, usePreviewMeme } from "@/hooks/galactica/meme/useMeme";
import { ReactElement, useContext } from "react";
import { MemeFilter } from "./meme.type";
import { Wallet, WalletUtils } from "@/components/ClientRoot";

interface MemeTimelineProps {
  filter?: MemeFilter;
}

// Default to load latest approved memes
const MemeTimeline = ({
  filter = {
    status: "APPROVED",
  },
}: MemeTimelineProps): ReactElement<MemeTimelineProps> => {
  const userWallet = useContext(Wallet);
  const { connect } = useContext(WalletUtils);

  const { memes, loadMore, isLoading, hasNext, loadInit } = useLatestMeme(
    filter,
    { requireInit: true }
  );
  const { memes: previewMemes } = usePreviewMeme();

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

  return (
    <>
      <div className="mt-4">
        <MemeInput />
      </div>
      <div className="px-2">
        {!userWallet.wallet ? (
          <>
            <MemeList
              memes={previewMemes}
              loadMore={() => {}}
              isLoading={false}
            />
            <div className="fixed bottom-0 left-0 w-full bg-corange backdrop-blur-lg bg-opacity-50 mt-8 md:mt-12">
              <div className="w-full flex items-center justify-center px-4 p-8 space-x-1 text-xl cursor-pointer">
                <div
                  className="underline text-blue-700 hover:text-blue-900 cursor-pointer disabled:cursor-not-allowed"
                  onClick={connect}
                >
                  Sign in
                </div>
                <span>to see more memes</span>
              </div>
            </div>
          </>
        ) : (
          <>
            {/* {userMemes.map((meme, i) => (
              <MemeListItem {...meme} isServerMeme={false} key={i} />
            ))} */}
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
