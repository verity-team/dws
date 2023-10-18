import Image from "next/image";
import BannerSection from "./BannerSection";

const Banner = () => {
  return (
    <div>
      <section className="grid grid-cols-3 mx-8 px-24 pt-8">
        <div className="col-span-2 flex h-full items-start justify-start">
          <h1 className="text-6xl leading-loose-2xl italic tracking-wide inline-block break-words">
            Let&#x27;s memefy counterculture!
            <br />
            Join the&nbsp;
            <span className="text-cred">Meme</span>&nbsp; ®Evolution.
          </h1>
        </div>
        <div className="col-span-1 flex items-center justify-center mx-8">
          <Image
            src="/images/wojak.png"
            alt="sad guy with hoodie"
            width={269}
            height={0}
            className="object-fill"
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
