import Image from "next/image";
import { SendIcon } from "lucide-react";
import XIcon from "@/components/icons/XIcon";

const Banner = () => {
  return (
    <div className="grid grid-cols-1 md:grid-cols-5 md:items-center md:mx-12">
      <section className="mx-8 md:col-span-3">
        <div>
          <h1 className="text-3xl italic tracking-wide inline-block break-words mt-2 md:text-5xl">
            We memefy counterculture!
            <br />
          </h1>
        </div>
        <div>
          <h1 className="text-3xl italic tracking-wide inline-block break-words mt-2 md:text-5xl">
            Join the&nbsp;
            <span className="text-cred">Meme</span>&nbsp; ®Evolution.
          </h1>
        </div>
        <div>
          <h2 className="text-2xl inline-block break-words mt-4 md:text-4xl">
            It&apos;s gonna to be Lit 🔥
          </h2>
        </div>
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
      </section>
      <div className="relative w-full h-[30vh] my-8 md:col-span-2 md:h-[50vh]">
        <Image
          src="/dws-images/banner1.png"
          alt="soyboy"
          fill
          sizes="100vw"
          className="object-contain overflow-hidden"
        />
      </div>
    </div>
  );
};

export default Banner;
