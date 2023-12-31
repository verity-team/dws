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
      <div className="mt-12">
        <Banner />
      </div>

      <div className="md:mt-12 md:max-w-4xl md:mx-auto xl:max-w-5xl">
        <StoryBanner />
      </div>

      <div className="mx-8 md:flex md:items-start md:justify-center">
        <LandingClientRoot>
          <Donate />
          <div className="mt-8 md:mt-12">
            <UserDonationStat />
          </div>
        </LandingClientRoot>
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
