import Image from "next/image";
import { BookOpenText } from "lucide-react";

const StoryBanner = () => {
  return (
    <section className="mx-8">
      <div>
        <h1 className="text-3xl italic tracking-wide inline-block break-words mt-2">
          Uniting the <span className="text-cred">Truth</span> movements
        </h1>
      </div>
      <div>
        <h2 className="text-2xl tracking-wide inline-block break-words mt-2">
          Media is failing, <span className="text-cred">Memes</span> are
          winning; but we can only truly win together.
        </h2>
      </div>
      <div className="mt-2">
        <h2 className="text-2xl tracking-wide inline-block break-words mt-1">
          Let&apos;s join forces!
        </h2>
      </div>
      <div className="mt-4 space-y-4">
        <button
          type="button"
          className="px-6 py-2 bg-cred text-white rounded-3xl border-4 border-black text-2xl tracking-wide uppercase"
        >
          <div className="flex items-center justify-center space-x-2">
            <p>Story</p>
            <BookOpenText />
          </div>
        </button>
      </div>
      <div className="relative w-full h-[30vh] my-8 p-2 box-border border-4 border-black bg-white">
        <div className="w-full h-full relative p-2 border-4 border-black">
          <Image
            src="/images/banner2.gif"
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
