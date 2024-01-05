import Image from "next/image";
import { BookOpenText, UserRound, UsersRound } from "lucide-react";

const CommunityBanner = () => {
  return (
    <section className="mx-8">
      <div>
        <h1 className="text-3xl italic tracking-wide inline-block break-words mt-2">
          <span className="text-cred">Memes</span> Creators United
        </h1>
      </div>
      <div>
        <h2 className="text-2xl tracking-wide inline-block break-words mt-2">
          Join our growing <span className="text-cred">Meme</span> Community
          with followers in the millions
        </h2>
      </div>
      <button
        type="button"
        className="mt-4 px-6 py-2 bg-cred text-white rounded-3xl border-4 border-black text-2xl tracking-wide uppercase"
      >
        <div className="flex items-center justify-center space-x-2">
          <p>Community</p>
          <UsersRound />
        </div>
      </button>
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

export default CommunityBanner;
