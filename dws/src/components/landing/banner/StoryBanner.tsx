import Image from "next/image";
import { SendIcon } from "lucide-react";
import XIcon from "@/components/icons/XIcon";

const StoryBanner = () => {
  return (
    <section className="mx-8">
      <h1 className="text-3xl italic tracking-wide inline-block break-words mt-2 md:text-5xl md:mt-8">
        The <span className="text-cred">Presale</span> is only the beginning...
      </h1>
      <h2 className="text-2xl tracking-wide inline-block break-words mt-2 md:text-4xl md:mt-8">
        Media is failing, <span className="text-cred">Memes</span> are winning;
        but we can only truly win together. Let&apos;s join forces!
      </h2>
      <div className="mt-4 space-y-4 md:flex md:items-center md:mt-8 md:space-x-4 md:space-y-0">
        <div>
          <button
            type="button"
            className="px-6 py-2 bg-cred text-white rounded-3xl border-4 border-black text-2xl tracking-wide uppercase"
          >
            <a
              className="flex items-center justify-center space-x-2 hover:text-gray-200"
              href="https://t.me/truthmemesofficialchat"
            >
              <p>Join Telegram</p>
              <SendIcon />
            </a>
          </button>
        </div>
        <div>
          <button
            type="button"
            className="px-6 py-2 bg-cred text-white rounded-3xl border-4 border-black text-2xl tracking-wide uppercase"
          >
            <a
              className="flex items-center justify-center space-x-2 hover:text-gray-200"
              href="https://twitter.com/_truthmemes_"
            >
              <p>Follow us</p>
              <XIcon />
            </a>
          </button>
        </div>
      </div>
      <div className="relative w-full h-[30vh] my-8 p-2 box-border border-8 border-black rounded-lg bg-white mx-auto md:h-[50vh]">
        <div className="w-full h-full relative p-2 border-4 border-black">
          <Image
            src="/dws-images/banner2.gif"
            alt="pepe army"
            fill
            sizes="100vw"
            className="object-cover overflow-hidden"
          />
        </div>
      </div>
    </section>
  );
};

export default StoryBanner;
