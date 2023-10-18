import ClientRoot from "@/components/ClientRoot";
import Banner from "@/components/landing/banner/Banner";
import LaunchTimer from "@/components/landing/banner/LaunchTimer";
import Navbar from "@/components/landing/navbar/Navbar";

export default function Home() {
  return (
    <>
      <ClientRoot>
        <Navbar />
        <div className="mx-8">
          <div className="px-8 pt-10">
            <LaunchTimer />
          </div>
        </div>
      </ClientRoot>

      <Banner />
    </>
  );
}
