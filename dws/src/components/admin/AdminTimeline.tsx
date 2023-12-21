import { ReactElement } from "react";
import { MemeFilter } from "../galactica/meme/meme.type";
import { useLatestMeme } from "@/hooks/galactica/meme/useMeme";
import MemeList from "../galactica/meme/list/MemeList";

interface AdminTimelineProps {
  filter?: MemeFilter;
}

const AdminTimeline = ({
  filter = {
    status: "PENDING",
  },
}: AdminTimelineProps): ReactElement<AdminTimelineProps> => {
  const { memes, hasNext, loadMore, isLoading } = useLatestMeme(filter);

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
      <MemeList memes={memes} loadMore={handleLoadMore} isLoading={isLoading} />
      {isLoading && (
        <div className="py-4 flex items-center justify-center">
          Loading more...
        </div>
      )}
    </>
  );
};

export default AdminTimeline;
