import { ReactElement, useEffect, useRef, useState } from "react";
import { MemeFilter } from "../meme/meme.type";
import { MemeUpload } from "@/api/galactica/meme/meme.type";
import { useLatestMeme } from "@/hooks/galactica/meme/useMeme";
import { CircularProgress } from "@mui/material";
import InfiniteScroll from "react-infinite-scroll-component";

interface MemeTimelineItemProps {
  meme: MemeUpload;
  onEndReached?: () => void;
}

interface MemeTimelineProps {
  filter: MemeFilter;
  ItemLayout: (props: { meme: MemeUpload }) => ReactElement;
}

const MemeTimeline = ({
  filter,
  ItemLayout,
}: MemeTimelineProps): ReactElement<MemeTimelineProps> => {
  const { data, ended, loadMore } = useLatestMeme(filter);

  return (
    <div className="w-full">
      <InfiniteScroll
        dataLength={data.length}
        next={loadMore}
        hasMore={!ended}
        loader={
          <div className="w-full flex flex-col items-center justify-center p-8">
            <CircularProgress />
            <div className="mt-4">Loading...</div>
          </div>
        }
        endMessage={
          <p className="text-center p-8 text-xl">
            <b>Yay! You have seen it all</b>
          </p>
        }
      >
        {data.map((meme) => (
          <ItemLayout key={meme.fileId} meme={meme} />
        ))}
      </InfiniteScroll>
    </div>
  );
};

export default MemeTimeline;
