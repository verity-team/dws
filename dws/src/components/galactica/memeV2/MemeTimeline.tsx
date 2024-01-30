import { ReactElement, useEffect, useRef } from "react";
import { MemeFilter } from "../meme/meme.type";
import { MemeUpload } from "@/api/galactica/meme/meme.type";
import { useLatestMeme } from "@/hooks/galactica/meme/useMeme";
import { CircularProgress } from "@mui/material";

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
  const { memes, isLoading, loadMore } = useLatestMeme(filter);

  const listEndFlag = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (!entries[0] || !entries[0].isIntersecting) {
          return;
        }

        loadMore();
      },
      { threshold: 1 }
    );

    const end = listEndFlag.current;
    if (!end) {
      return;
    }

    observer.observe(end);
    return () => {
      observer.unobserve(end);
    };

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [listEndFlag]);

  return (
    <div className="w-full">
      <div className="space-y-4">
        {memes.map((meme) => (
          <ItemLayout key={meme.fileId} meme={meme} />
        ))}
        {isLoading && (
          <div>
            <CircularProgress />
            <div>Loading</div>
          </div>
        )}
        {memes.length > 0 && <div ref={listEndFlag}></div>}
      </div>
    </div>
  );
};

export default MemeTimeline;
