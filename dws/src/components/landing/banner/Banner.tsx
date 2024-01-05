import Image from "next/image";
import BannerSection from "./BannerSection";

const Banner = () => {
  return (
    <div>
      <section className="grid grid-cols-1 md:grid-cols-3 mx-8 px-24 py-24 relative">
        <div className="md:col-span-2 flex h-full items-start justify-start">
          <h1 className="text-5xl leading-loose-2xl italic tracking-wide inline-block break-words">
            Let&#x27;s memefy counterculture!
            <br />
            Join the&nbsp;
            <span className="text-cred">Meme</span>&nbsp; ®Evolution.
          </h1>
        </div>
        <div className="md:col-span-1 flex items-center justify-center mx-8 w-full h-full relative">
          <Image
            src="/images/banner1.png"
            alt="soyboy"
            fill
            sizes="100vw"
            className="object-contain overflow-visible"
          />
        </div>
      </section>
      <BannerSection className="bg-cgreen">
        <h1 className="text-6xl leading-loose-2xl italic">
          Uniting and tokenizing the truth movement.
        </h1>
        <h1 className="text-4xl leading-loose-xl my-6 break-words tracking-wide">
          Bringing together the two most vibrant and viral communities around
          truth-seeking: Meme-Artists and Crypto-Freedom-Lovers.
        </h1>
      </BannerSection>
      <BannerSection className="bg-white">
        <h1 className="text-6xl leading-loose-2xl italic">Along with...</h1>
        <h1 className="text-4xl leading-loose-xl my-6 break-words tracking-wide">
          Journalists, Hackers, Creatives, ...
          <br />
          <br />
          Let&#x27;s work towards sustaining free and open societies &amp; have
          fun while doing it!
        </h1>
      </BannerSection>
    </div>
  );
};

export default Banner;
