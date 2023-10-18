import ClientRoot from "@/components/ClientRoot";
import Banner from "@/components/landing/banner/Banner";
import MemeSlideshow from "@/components/landing/carousel/MemeSlideshow";
import Navbar from "@/components/landing/navbar/Navbar";
import dynamic from "next/dynamic";

const LaunchTimer = dynamic(
  () => import("@/components/landing/banner/LaunchTimer"),
  { ssr: false }
);

export default function Home() {
  return (
    <>
      <ClientRoot>
        <Navbar />
        <div className="mx-8">
          <div className="px-24 pt-10">
            <LaunchTimer />
          </div>
        </div>
      </ClientRoot>
      <Banner />
      <MemeSlideshow />
    </>
  );
}
