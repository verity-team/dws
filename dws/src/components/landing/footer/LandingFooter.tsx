import DiscordIcon from "@/components/icons/DiscordIcon";
import TelegramIcon from "@/components/icons/TelegramIcon";
import XIcon from "@/components/icons/XIcon";
import { InstagramIcon, SendIcon, TwitterIcon } from "lucide-react";
import Image from "next/image";

const LandingFooter = () => {
  return (
    <>
      <div className="mx-8">
        <div className="md:flex md:items-center md:justify-between">
          <div>
            <div>
              <h1 className="text-3xl italic tracking-wide inline-block break-words md:text-5xl">
                Join the the fun on <span className="text-cred">Telegram</span>
              </h1>
            </div>
            <button
              type="button"
              className="mt-4 px-6 py-2 bg-cred text-white rounded-3xl border-4 border-black text-2xl tracking-wide uppercase"
            >
              <div className="flex items-center justify-center space-x-2">
                <p>Join Telegram</p>
                <SendIcon />
              </div>
            </button>
          </div>
          <div className="relative md:flex md:items-center md:justify-center">
            <Image
              src="/images/wojak.png"
              alt="soyboy"
              width={300}
              height={300}
              className="object-contain overflow-hidden"
            />
          </div>
        </div>
        <div className="my-8 md:flex md:items-center md:justify-between">
          <div className="flex items-center justify-center">
            <Image
              src="/images/logo.png"
              alt="eye of truth"
              width={48}
              height={0}
              className="h-auto object-contain"
            />
            <div className="text-xl uppercase">Truth memes</div>
          </div>
          <div className="flex items-center justify-center space-x-4">
            <XIcon width={48} height={48} fill="#000" stroke="#000" />
            <TelegramIcon width={48} height={48} />
          </div>
        </div>
      </div>
      {/* <div className="w-full grid grid-cols-4 mt-8">
        <div className="border border-black flex flex-col items-center justify-center p-4 bg-corange cursor-pointer hover:bg-orange-400">
          <TwitterIcon />
          <div className="mt-1">Twitter</div>
        </div>
        <div className="border border-black flex flex-col items-center justify-center p-4 bg-corange cursor-pointer hover:bg-orange-400">
          <SendIcon />
          <div>Telegram</div>
        </div>
        <div className="border border-black flex flex-col items-center justify-center p-4 bg-corange cursor-pointer hover:bg-orange-400">
          <DiscordIcon />
          <div>Discord</div>
        </div>
        <div className="border border-black flex flex-col items-center justify-center p-4 bg-corange cursor-pointer hover:bg-orange-400">
          <InstagramIcon />
          <div>Instagram</div>
        </div>
      </div> */}
    </>
  );
};

export default LandingFooter;
