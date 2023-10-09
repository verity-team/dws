import { ReactElement, memo } from "react";
import { AvailableToken } from "./Donate";

interface TokenSelectorProps {
  selectedToken: AvailableToken;
  onTokenChange: (token: AvailableToken) => void;
}

const avaiableTokens: Array<AvailableToken> = ["ETH", "USDT"];

const TokenSelector = ({
  selectedToken,
  onTokenChange,
}: TokenSelectorProps): ReactElement<TokenSelectorProps> => {
  return (
    <>
      {avaiableTokens.map((token) => (
        <div key={token} className="py-1">
          <input
            type="radio"
            name="token"
            value={token}
            checked={selectedToken === token}
            onChange={() => onTokenChange(token)}
          />
          <label>{token}</label>
        </div>
      ))}
    </>
  );
};

export default memo(TokenSelector);
