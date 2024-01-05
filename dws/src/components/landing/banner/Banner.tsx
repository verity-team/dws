import Image from "next/image";
import BannerSection from "./BannerSection";
import { TwitterIcon, Teleg, SendIcon } from "lucide-react";

const Banner = () => {
  return (
    <>
      <section className="mx-8">
        <div>
          <h1 className="text-3xl italic tracking-wide inline-block break-words mt-2">
            We memefy counterculture!
            <br />
          </h1>
        </div>
        <div>
          <h1 className="text-3xl italic tracking-wide inline-block break-words mt-2">
            Join the&nbsp;
            <span className="text-cred">Meme</span>&nbsp; ®Evolution.
          </h1>
        </div>
        <div>
          <h2 className="text-2xl inline-block break-words mt-4">
            It&apos;s gonna to be Lit 🔥
          </h2>
        </div>
        <div className="mt-4 space-y-4">
          <div>
            <button
              type="button"
              className="px-6 py-2 bg-cred text-white rounded-3xl border-4 border-black text-2xl tracking-wide uppercase"
            >
              <div className="flex items-center justify-center space-x-2">
                <p>Join Telegram</p>
                <SendIcon />
              </div>
            </button>
          </div>
          <div>
            <button
              type="button"
              className="px-6 py-2 bg-cred text-white rounded-3xl border-4 border-black text-2xl tracking-wide uppercase"
            >
              <div className="flex items-center justify-center space-x-2">
                <p>Follow us</p>
                <TwitterIcon />
              </div>
            </button>
          </div>
        </div>
      </section>
      <div className="relative w-full h-[30vh] my-8">
        <Image
          src="/images/banner1.png"
          alt="soyboy"
          fill
          sizes="100vw"
          className="object-contain overflow-hidden"
        />
      </div>
    </>
  );
};

export default Banner;
