"use client";

import { getMemeImage, getSingleMeme } from "@/api/galactica/meme/meme";
import { MemeUpload } from "@/api/galactica/meme/meme.type";
import { roboto } from "@/app/fonts";
import ItemToolbar from "@/components/galactica/meme/list/toolbar/ItemToolbar";
import { Maybe } from "@/utils";
import { getTimeElapsedString } from "@/utils/utils";
import { getWalletShorthand } from "@/utils/wallet/wallet";
import { CircularProgress } from "@mui/material";
import Avatar from "boring-avatars";
import Image from "next/image";
import { ReactElement, useEffect, useState } from "react";
import ArrowBackIcon from "@mui/icons-material/ArrowBack";
import { redirect } from "next/navigation";

interface SingleMemePageProps {
  params: {
    id: string;
  };
}

const SingleMemePage = ({
  params,
}: SingleMemePageProps): ReactElement<SingleMemePageProps> => {
  const [memeData, setMemeData] = useState<Maybe<MemeUpload>>(null);
  const [memeImage, setMemeImage] = useState<string>("");

  useEffect(() => {
    const loadInit = async () => {
      const data = await getSingleMeme(params.id);
      if (data == null) {
        throw new Error("Cannot find meme");
      }

      const memeImage = await getMemeImage(data.fileId);

      setMemeData(data);
      setMemeImage(memeImage ?? "");
    };

    loadInit();
  }, [params]);

  if (memeData == null) {
    return (
      <div className="w-full h-screen flex items-center justify-center">
        <CircularProgress size={100} />
      </div>
    );
  }

  const { userId, createdAt, caption } = memeData;

  return (
    <div className="grid grid-cols-12">
      <div className="col-span-3"></div>
      <div className="col-span-6">
        <div className={roboto.className}>
          <a
            className="flex items-center my-8 hover:text-blue-500 hover:underline cursor-pointer"
            href="/meme"
          >
            <ArrowBackIcon fontSize="medium" />
            <p className="text-lg">Back to timeline</p>
          </a>
          <div className="w-full mt-6 border-t border-b border-gray-100">
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
                width={0}
                height={0}
                className="object-contain w-auto max-w-[80%] max-h-[40vh]"
                alt={caption}
                priority={true}
              />
            </div>
            <div className="mt-2">
              <ItemToolbar />
            </div>
          </div>
        </div>
      </div>
      <div className="col-span-3"></div>
    </div>
  );
};

export default SingleMemePage;
