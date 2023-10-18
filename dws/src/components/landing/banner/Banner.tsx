import Image from "next/image";

const Banner = () => {
  return (
    <div className="mx-8">
      <div className="grid grid-cols-3 px-24 pt-8">
        <div className="col-span-2 items-baseline justify-start">
          <h1 className="text-6xl leading-[80px] italic tracking-wide ">
            Let&#x27;s memefy counterculture!
            <br />
            Join the&nbsp;
            <span className="text-cred">Meme</span>
            ®Evolution.
          </h1>
        </div>
        <div className="col-span-1 flex items-center justify-center">
          <Image
            src="/images/wojak.png"
            alt="sad guy with hoodie"
            width={269}
            height={0}
            className="max-w-full h-auto pr-4"
          />
        </div>
      </div>
    </div>
  );
};

export default Banner;
