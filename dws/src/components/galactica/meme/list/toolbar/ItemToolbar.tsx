"use client";

import ThumbUpIconOutlined from "@mui/icons-material/ThumbUpOutlined";
import { ReactElement, SyntheticEvent, useCallback } from "react";
import toast from "react-hot-toast";
import ShareButton from "./ShareButton";

interface ItemToolbarProps {
  memeId: string;
}

// TODO: Separate actions into different components when features other than "Share" are ready
const ItemToolbar = ({
  memeId,
}: ItemToolbarProps): ReactElement<ItemToolbarProps> => {
  const handleLikeClick = useCallback((event: SyntheticEvent) => {
    event.stopPropagation();
    toast("Coming soon...", { icon: "🚧" });
  }, []);

  return (
    <div className="grid grid-cols-3">
      <button
        type="button"
        className="flex items-center justify-center px-4 py-2 rounded-lg hover:bg-slate-200 m-1"
        onClick={handleLikeClick}
      >
        <ThumbUpIconOutlined className="text-2xl" />
        <div className="ml-1.5 text-lg font-changa">Like</div>
      </button>
      <div></div>
      <ShareButton memeId={memeId} />
    </div>
  );
};

export default ItemToolbar;
