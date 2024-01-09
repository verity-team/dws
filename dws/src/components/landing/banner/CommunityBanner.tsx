import { BookOpenText, UsersRound } from "lucide-react";

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
      <div className="mt-4 md:flex md:items-center md:mt-8 md:space-x-4 md:space-y-0">
        <div>
          <button
            type="button"
            className="mt-4 px-6 py-2 bg-cred text-white rounded-3xl border-4 border-black text-2xl tracking-wide uppercase md:mt-8"
          >
            <a
              className="flex items-center justify-center space-x-2 hover:text-gray-200"
              href="/community.html"
            >
              <p>Community</p>
              <UsersRound />
            </a>
          </button>
        </div>
        <div>
          <button
            type="button"
            className="mt-4 px-6 py-2 bg-cred text-white rounded-3xl border-4 border-black text-2xl tracking-wide uppercase md:mt-8"
          >
            <a
              className="flex items-center justify-center space-x-2 hover:text-gray-200"
              href="/story.html"
            >
              <p>Story</p>
              <BookOpenText />
            </a>
          </button>
        </div>
      </div>
    </section>
  );
};

export default CommunityBanner;
