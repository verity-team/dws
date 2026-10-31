import "server-only";

import { getSingleMeme } from "@/api/galactica/meme/meme";
import { roboto } from "@/app/fonts";
import ItemToolbar from "@/components/galactica/meme/list/toolbar/ItemToolbar";
import { getTimeElapsedString } from "@/utils/utils";
import { getWalletShorthand } from "@/utils/wallet/wallet";
import Avatar from "boring-avatars";
import Image from "next/image";
import { ReactElement, cache } from "react";
import ArrowBackIcon from "@mui/icons-material/ArrowBack";
import { Metadata } from "next";
import { Toaster } from "react-hot-toast";

interface SingleMemePageProps {
  params: {
    id: string;
  };
}

// One galactica request per page render, shared by the page and its metadata
const getMeme = cache(getSingleMeme);

const getMemeImageUrl = (fileId: string): string =>
  `${process.env.NEXT_PUBLIC_GALACTICA_API_URL}/meme/image/${encodeURIComponent(
    fileId
  )}`;

// Built per request: a shared metadata object would leak one meme's card into
// a concurrent request for another
export const generateMetadata = async ({
  params,
}: SingleMemePageProps): Promise<Metadata> => {
  const memeData = await getMeme(params.id);

  return {
    twitter: {
      card: "summary_large_image",
      description:
        "Join the Truthmemes army and deliver the truth to everyone!",
      site: process.env.HOST_URL,
      ...(memeData != null && {
        title: memeData.caption,
        images: [getMemeImageUrl(memeData.fileId)],
      }),
    },
  };
};

const SingleMemePage = async ({
  params,
}: SingleMemePageProps): Promise<ReactElement<SingleMemePageProps>> => {
  const memeData = await getMeme(params.id);
  if (memeData == null) {
    throw new Error("Cannot find meme");
  }

  const { fileId, userId, createdAt, caption } = memeData;

  const memeImage = getMemeImageUrl(fileId);

  return (
    <div className="grid grid-cols-12 px-8">
      <div className="col-span-12 md:col-span-4 md:col-start-5 md:mt-12">
        <div className={roboto.className}>
          <a
            className="flex items-center my-8 hover:text-blue-500 hover:underline cursor-pointer space-x-4"
            href="/meme"
          >
            <ArrowBackIcon fontSize="medium" />
            <p className="text-lg">Back to timeline</p>
          </a>
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
            <div className="flex items-center justify-center mt-4">
              <Image
                src={memeImage}
                width={1000}
                height={1000}
                className="object-contain w-auto max-w-[80%] max-h-[40vh]"
                alt={caption}
                priority={true}
              />
            </div>
            <div className="mt-2">
              <ItemToolbar memeId={fileId} />
            </div>
          </div>
        </div>
      </div>
      <Toaster />
    </div>
  );
};

export default SingleMemePage;
