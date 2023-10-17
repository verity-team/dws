import ClientRoot from "@/components/ClientRoot";
import Navbar from "@/components/landing/navbar/Navbar";
import DemoConnect from "@/components/metamask/DemoConnect";

export default function Home() {
  return (
    <>
      <Navbar />
      <ClientRoot>
        <DemoConnect />
      </ClientRoot>
    </>
  );
}
