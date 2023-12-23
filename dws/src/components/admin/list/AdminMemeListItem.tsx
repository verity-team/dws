import { useMemeImage } from "@/hooks/galactica/meme/useMeme";
import Image from "next/image";
import { ReactElement, useEffect, useMemo } from "react";
import Avatar from "boring-avatars";
import { getWalletShorthand } from "@/utils/wallet/wallet";
import { getTimeElapsedString } from "@/utils/utils";
import { roboto } from "@/app/fonts";
import AdminToolbar from "./AdminMemeToolbar";
import { MemeUploadStatus } from "@/components/galactica/meme/meme.type";

interface AdminMemeListItemProps {
  userId: string;
  fileId: string;
  caption: string;
  createdAt: string;
  status: MemeUploadStatus;
  isServerMeme: boolean;
}

const AdminMemeListItem = ({
  userId,
  fileId,
  caption,
  createdAt,
  status,
  isServerMeme,
}: AdminMemeListItemProps): ReactElement<AdminMemeListItemProps> => {
  const { url, loading, getMemeImage } = useMemeImage();

  useEffect(() => {
    // No need to query image from server if it's not a server meme
    if (!isServerMeme) {
      return;
    }

    getMemeImage(fileId);
  }, []);

  const dateDescription = useMemo(() => {
    return getTimeElapsedString(createdAt);
  }, [createdAt]);

  if (loading || !url) {
    return (
      <div className="p-12 flex items-center justify-center">Loading...</div>
    );
  }

  return (
    <div className={roboto.className}>
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
            src={isServerMeme ? url : fileId}
            width={0}
            height={0}
            className="object-contain w-auto max-w-[80%] max-h-[40vh]"
            alt={caption}
          />
        </div>
        <AdminToolbar memeStatus={status} />
      </div>
    </div>
  );
};

export default AdminMemeListItem;
