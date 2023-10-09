import ClientRoot from "@/components/ClientRoot";
import DemoConnectButton from "@/components/metamask/DemoConnectButton";

export default function Home() {
  return (
    <>
      <div>Welcome to DWS. Testing...</div>
      <ClientRoot>
        <DemoConnectButton />
      </ClientRoot>
    </>
  );
}
