import ClientRoot from "@/components/ClientRoot";
import Navbar from "@/components/landing/navbar/Navbar";
import DemoConnect from "@/components/metamask/DemoConnect";
import { theme } from "@/utils/theme";
import { ThemeProvider } from "@mui/material";

export default function Home() {
  return (
    <ClientRoot>
      <Navbar />
      <DemoConnect />
    </ClientRoot>
  );
}
