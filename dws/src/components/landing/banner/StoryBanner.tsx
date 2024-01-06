import Image from "next/image";
import { BookOpenText } from "lucide-react";

const StoryBanner = () => {
  return (
    <section className="mx-8">
      <h1 className="text-3xl italic tracking-wide inline-block break-words mt-2 md:text-5xl md:mt-8">
        Uniting the <span className="text-cred">Truth</span> movements
      </h1>
      <h2 className="text-2xl tracking-wide inline-block break-words mt-2 md:text-4xl md:mt-8">
        Media is failing, <span className="text-cred">Memes</span> are winning;
        but we can only truly win together.
      </h2>
      <h2 className="mt-1 text-2xl tracking-wide inline-block break-words md:text-4xl md:mt-4">
        Let&apos;s join forces!
      </h2>
      <div className="mt-4 space-y-4 md:mt-8">
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
      <div className="relative w-full h-[30vh] my-8 p-2 box-border border-8 border-black rounded-lg bg-white md:w-1/2 md:aspect-square mx-auto">
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
