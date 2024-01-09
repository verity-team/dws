import Banner from "@/components/landing/banner/Banner";
import MemeSlideshow from "@/components/landing/carousel/MemeSlideshow";
import LandingFooter from "@/components/landing/footer/LandingFooter";
import Navbar from "@/components/landing/navbar/Navbar";
import Newsletter from "@/components/landing/newsletter/Newsletter";
import dynamic from "next/dynamic";
import StoryBanner from "@/components/landing/banner/StoryBanner";
import CommunityBanner from "@/components/landing/banner/CommunityBanner";
import DonateForm from "@/components/ico/donatev2/form/DonateForm";
import LandingClientRoot from "@/components/landing/LandingClientRoot";
import Donate from "@/components/ico/donatev2/Donate";
import DonationStat from "@/components/ico/stats/donation/DonationStat";
import UserStat from "@/components/ico/stats/user/UserStat";
import UserDonationStat from "@/components/ico/stats/user/UserDonationStat";
import AFCForm from "@/components/ico/donatev2/AFCForm";
import Image from "next/image";

const LaunchTimer = dynamic(
  () => import("@/components/landing/banner/LaunchTimer"),
  { ssr: false }
);

export default function Home() {
  return (
    <>
      <Navbar />
      {/* <div className="mx-8">
        <div className="px-24 pt-10">
          <LaunchTimer />
        </div>
      </div> */}

      <div className="mt-12 mx-8 grid grid-cols-1 md:grid-cols-2">
        <LandingClientRoot>
          <div className="md:flex md:flex-col md:items-center md:justify-start md:origin-top md:scale-125">
            <Donate />
            {/* Share section */}
            <div className="mt-4">
              <AFCForm />
            </div>
            <div className="text-center">
              <a
                className="text-base tracking-wide inline-block break-words mt-2 hover:underline hover:text-blue-600 cursor-pointer"
                href="/terms.html"
              >
                <span className="text-cred">$TRUTH</span> Terms & Conditions
              </a>
            </div>
          </div>
          <div className="relative h-[90vh] hidden md:block">
            <Image
              src="/images/point.png"
              alt="guy pointing at something"
              fill
              className="object-contain w-full h-full"
            />
            <div className="w-full h-full flex items-end justify-end mt-12">
              <h1 className="text-3xl italic tracking-wide inline-block break-words mt-2 md:text-5xl md:mr-12">
                <span className="text-cred">$TRUTH</span> shall be revealed
              </h1>
            </div>
          </div>
          <div
            className="mt-8 md:mt-12 md:col-span-2 md:flex md:items-center md:justify-center md:w-full"
            id="thank-you"
          >
            <UserDonationStat />
          </div>
        </LandingClientRoot>
      </div>

      {/* <div className="md:mt-12 md:max-w-4xl md:mx-auto xl:max-w-5xl">
        <Banner />
      </div> */}

      <div className="mt-8 md:mt-12 md:max-w-4xl md:mx-auto xl:max-w-5xl">
        <StoryBanner />
      </div>

      <div className="mt-8 md:mt-12">
        <MemeSlideshow />
      </div>

      <div className="mt-8 md:mt-12 md:max-w-4xl md:mx-auto xl:max-w-6xl">
        <CommunityBanner />
      </div>

      <div className="mt-8 md:mt-12 md:mx-12 xl:max-w-6xl xl:mx-auto">
        <Newsletter />
      </div>

      <div className="mt-8">
        <LandingFooter />
      </div>
    </>
  );
}
