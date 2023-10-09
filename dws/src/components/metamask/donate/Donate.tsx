import { ReactElement, useCallback, useState } from "react";
import TokenSelector from "./TokenSelector";
import DonateForm from "./DonateForm";

export type AvailableToken = "ETH" | "USDT";

interface DonateProps {
  account: string;
}

const Donate = ({ account }: DonateProps): ReactElement<DonateProps> => {
  const [selectedToken, setSelectedToken] = useState<AvailableToken>("ETH");

  const handleTokenChange = useCallback((token: AvailableToken) => {
    setSelectedToken(token);
  }, []);

  return (
    <>
      <TokenSelector
        selectedToken={selectedToken}
        onTokenChange={handleTokenChange}
      />
      <DonateForm selectedToken={selectedToken} account={account} />
    </>
  );
};

export default Donate;
