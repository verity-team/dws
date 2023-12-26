"use client";

import { useMemeImage } from "@/hooks/galactica/meme/useMeme";
import Image from "next/image";
import { ReactElement, SyntheticEvent, useEffect, useMemo } from "react";
import Avatar from "boring-avatars";
import { getWalletShorthand } from "@/utils/wallet/wallet";
import { getTimeElapsedString } from "@/utils/utils";
import { roboto } from "@/app/fonts";
import ItemToolbar from "./toolbar/ItemToolbar";
import { useRouter } from "next/navigation";

interface MemeListItemProps {
  userId: string;
  fileId: string;
  caption: string;
  createdAt: string;
  isServerMeme: boolean;
}

const MemeListItem = ({
  userId,
  fileId,
  caption,
  createdAt,
  isServerMeme,
}: MemeListItemProps): ReactElement<MemeListItemProps> => {
  const { url, loading, getMemeImage } = useMemeImage();
  const router = useRouter();

  useEffect(() => {
    // No need to query image from server if it's not a server meme
    if (!isServerMeme) {
      return;
    }

    getMemeImage(fileId);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const dateDescription = useMemo(() => {
    return getTimeElapsedString(createdAt);
  }, [createdAt]);

  const handlePostClick = (event: SyntheticEvent) => {
    event.stopPropagation();
    router.push(`/meme/${fileId}`);
  };

  if (loading || !url) {
    return (
      <div className="p-12 flex items-center justify-center">Loading...</div>
    );
  }

  return (
    <div className={`${roboto.className} cursor-pointer`}>
      <div className="w-full mt-6 border-b border-gray-100">
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
        <div
          className="flex items-center justify-center mt-4 cursor-pointer"
          onClick={handlePostClick}
        >
          <Image
            src={isServerMeme ? url : fileId}
            width={0}
            height={0}
            className="object-contain w-auto max-w-[80%] max-h-[40vh]"
            alt={caption}
          />
        </div>
        <div className="mt-2">
          <ItemToolbar />
        </div>
      </div>
    </div>
  );
};

export default MemeListItem;
