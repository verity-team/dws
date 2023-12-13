import { MemeUpload } from "@/api/galactica/meme/meme.type";
import { useMemeImage } from "@/hooks/galactica/meme/useMeme";
import Image from "next/image";
import { ReactElement, useEffect, useMemo } from "react";
import Avatar from "boring-avatars";
import { getWalletShorthand } from "@/utils/wallet/wallet";
import { getTimeElapsedString } from "@/utils/utils";

interface MemeListItemProps {
  meme: MemeUpload;
}

const MemeListItem = ({
  meme,
}: MemeListItemProps): ReactElement<MemeListItemProps> => {
  const { url, loading, getMemeImage } = useMemeImage();

  useEffect(() => {
    getMemeImage(meme.fileId);
  }, []);

  const { userId, caption, createdAt } = meme;

  const dateDescription = useMemo(() => {
    return getTimeElapsedString(createdAt);
  }, [createdAt]);

  if (loading || !url) {
    return (
      <div className="p-12 flex items-center justify-center">Loading...</div>
    );
  }

  return (
    <div className="w-full py-8 border-t border-b border-gray-100">
      <div className="flex items-center justify-start space-x-4">
        <Avatar size={40} name={userId} variant="marble" />
        <div>
          <div className="font-semibold text-sm">
            {getWalletShorthand(userId)}
            <span className="text-sm font-light text-gray-500">
              {" "}
              • {dateDescription}
            </span>
          </div>
          {caption && <div className="text-base mt-0.5">{caption}</div>}
        </div>
      </div>
      <div className="flex items-center justify-center mt-4">
        <Image
          src={url}
          width={0}
          height={0}
          className="object-contain w-auto max-w-[80%] max-h-full"
          alt={meme.caption}
        />
      </div>
    </div>
  );
};

export default MemeListItem;
