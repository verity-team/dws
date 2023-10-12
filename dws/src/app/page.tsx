import ClientRoot from "@/components/ClientRoot";
import DemoConnect from "@/components/metamask/DemoConnect";

export default function Home() {
  return (
    <>
      <div>Welcome to DWS. Testing...</div>
      <ClientRoot>
        <DemoConnect />
      </ClientRoot>
    </>
  );
}
