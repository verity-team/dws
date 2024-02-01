import {
  requestApproveMeme,
  requestDeclineMeme,
  requestRevertMemeReview,
} from "@/api/galactica/admin/admin";
import { MemeUpload } from "@/api/galactica/meme/meme.type";
import { roboto } from "@/app/fonts";
import AdminToolbar from "@/components/admin/list/AdminMemeToolbar";
import { useMemeImage } from "@/hooks/galactica/meme/useMeme";
import useAccountId from "@/hooks/store/useAccountId";
import { getTimeElapsedString } from "@/utils/utils";
import { getWalletShorthand } from "@/utils/wallet/wallet";
import { CircularProgress } from "@mui/material";
import Avatar from "boring-avatars";
import Image from "next/image";
import toast from "react-hot-toast";

interface MemeItemProps {
  meme: MemeUpload;
}

const MemeItem = ({ meme }: MemeItemProps) => {
  const { imageUrl, isLoading } = useMemeImage(meme.fileId);

  const { userId, caption, createdAt, status, fileId } = meme;

  const { accessToken } = useAccountId();

  const handleApproveMeme = async () => {
    if (!accessToken) {
      return;
    }

    const succeed = await requestApproveMeme(accessToken, fileId);
    if (!succeed) {
      toast.error("Failed to approve meme");
      return;
    }

    toast.success("Meme approved");
  };

  const handleDeclineMeme = async () => {
    if (!accessToken) {
      return;
    }

    const succeed = await requestDeclineMeme(accessToken, fileId);
    if (!succeed) {
      toast.error("Failed to decline meme");
      return;
    }

    toast.success("Meme declined");
  };

  const handleRevertMemeStatus = async () => {
    if (!accessToken) {
      return;
    }

    const succeed = await requestRevertMemeReview(accessToken, fileId);
    if (!succeed) {
      toast.error("Failed to revert meme's status");
      return;
    }

    toast.success("Meme is now in the pending list");
  };

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
                • {getTimeElapsedString(createdAt)}
              </span>
            </div>
            {caption && <div className="text-base mt-0.5">{caption}</div>}
          </div>
        </div>
        {isLoading || !imageUrl ? (
          <div className="w-full flex flex-col items-center justify-center p-8">
            <CircularProgress />
            <div className="mt-2">Loading...</div>
          </div>
        ) : (
          <div className="flex items-center justify-center mt-4 cursor-pointer">
            <Image
              src={imageUrl}
              width={0}
              height={0}
              className="border-4 border-black box-border rounded-lg w-auto max-h-[40vh] object-contain"
              alt={caption}
            />
          </div>
        )}
        <AdminToolbar
          memeStatus={status}
          onApprove={handleApproveMeme}
          onDeny={handleDeclineMeme}
          onRevert={handleRevertMemeStatus}
        />
      </div>
    </div>
  );
};

export default MemeItem;
