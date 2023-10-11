import ClientRoot from "@/components/ClientRoot";
import DemoConnect from "@/components/metamask/DemoConnect";
import DonationStat from "@/components/stats/donation/DonationStat";

export const runtime = "edge";

export default function Home() {
  return (
    <>
      <div>Welcome to DWS. Testing...</div>
      <ClientRoot>
        <DemoConnect />
        <DonationStat />
      </ClientRoot>
    </>
  );
}
