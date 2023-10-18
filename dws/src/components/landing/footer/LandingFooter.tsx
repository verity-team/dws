import Image from "next/image";

const LandingFooter = () => {
  return (
    <div className="flex flex-col items-center justify-center md:flex-row md:items-center md:justify-between mx-16 my-8">
      <div className="flex">
        <Image
          src="/images/logo.png"
          alt="truth meme eye logo"
          width={20}
          height={0}
          className="max-w-[15px] h-auto object-contain"
        />
        <div className="text-sm uppercase leading-5 font-sans opacity-70 pl-2">
          TRUTH-MEMES BY verity.team ©2023
        </div>
      </div>
      <div className="flex flex-col space-x-0 mt-8 space-y-8 md:block md:space-y-0 md:space-x-12 md:mt-0">
        <a
          href="https://www.facebook.com/webflow/"
          target="_blank"
          className="text-xl uppercase font-bold text-gray-500"
        >
          TELEGRAM
        </a>
        <a
          href="https://www.facebook.com/webflow/"
          target="_blank"
          className="text-xl uppercase font-bold text-gray-500"
        >
          DISCORD
        </a>
        <a
          href="https://twitter.com/webflow"
          target="_blank"
          className="text-xl uppercase font-bold text-gray-500"
        >
          Twitter
        </a>
        <a
          href="https://www.instagram.com/webflowapp/"
          target="_blank"
          className="text-xl uppercase font-bold text-gray-500"
        >
          Instagram
        </a>
      </div>
    </div>
  );
};

export default LandingFooter;
