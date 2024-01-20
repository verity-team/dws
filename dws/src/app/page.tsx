import MemeSlideshow from "@/components/landing/carousel/MemeSlideshow";
import LandingFooter from "@/components/landing/footer/LandingFooter";
import Navbar from "@/components/landing/navbar/Navbar";
import Newsletter from "@/components/landing/newsletter/Newsletter";
import StoryBanner from "@/components/landing/banner/StoryBanner";
import CommunityBanner from "@/components/landing/banner/CommunityBanner";
import LandingClientRoot from "@/components/landing/LandingClientRoot";
import Donate from "@/components/ico/donatev2/Donate";
import AFCForm from "@/components/ico/donatev2/AFCForm";
import Image from "next/image";
import { CircularProgress } from "@mui/material";
import { Suspense } from "react";

export default function Home() {
  return (
    <>
      <Navbar />

      <Suspense
        fallback={
          <div className="absolute w-screen h-screen top-0 left-0 flex items-center justify-center bg-transparent">
            <CircularProgress />
          </div>
        }
      >
        <div className="mt-4 mx-8 grid grid-cols-1 md:grid-cols-2">
          <LandingClientRoot>
            <div className="md:flex md:flex-col md:items-center md:justify-start">
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
                <h1 className="text-3xl italic tracking-wide inline-block break-words mt-2 md:text-2xl xl:text-4xl xl:mr-12">
                  <span className="text-cred">$TRUTH</span> shall be revealed
                </h1>
              </div>
            </div>
          </LandingClientRoot>
        </div>
      </Suspense>
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
