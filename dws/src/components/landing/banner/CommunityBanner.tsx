import { UsersRound } from "lucide-react";

const CommunityBanner = () => {
  return (
    <section className="mx-8">
      <h1 className="text-3xl italic tracking-wide inline-block break-words mt-2 md:text-5xl md:mt-8">
        <span className="text-cred">Memes</span> Creators United
      </h1>
      <h2 className="text-2xl tracking-wide inline-block break-words mt-2 md:text-4xl md:mt-8">
        Join our growing <span className="text-cred">Meme</span> Community with
        followers in the millions
      </h2>
      <button
        type="button"
        className="mt-4 px-6 py-2 bg-cred text-white rounded-3xl border-4 border-black text-2xl tracking-wide uppercase md:mt-8"
      >
        <div className="flex items-center justify-center space-x-2">
          <p>Community</p>
          <UsersRound />
        </div>
      </button>
    </section>
  );
};

export default CommunityBanner;
