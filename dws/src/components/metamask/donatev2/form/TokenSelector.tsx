import { AvailableToken, avaiableTokens } from "@/utils/token";
import { ReactElement, memo } from "react";

interface TokenSelectorProps {
  selectedToken: AvailableToken;
  onTokenChange: (token: AvailableToken) => void;
}

const TokenSelector = ({
  selectedToken,
  onTokenChange,
}: TokenSelectorProps): ReactElement<TokenSelectorProps> => {
  return (
    <div className="grid grid-cols-3 gap-2 p-2">
      {avaiableTokens.map((token) => {
        if (token === selectedToken) {
          return (
            <button
              key={token}
              className="w-full py-2 rounded-2xl border-2 border-black bg-white text-center text-gray-900 cursor-pointer"
              onClick={() => onTokenChange(token)}
            >
              {token}
            </button>
          );
        }

        return (
          <button
            key={token}
            className="w-full py-2 rounded-2xl border-2 border-gray-400 bg-white text-center text-gray-400 cursor-pointer"
            onClick={() => onTokenChange(token)}
          >
            {token}
          </button>
        );
      })}
    </div>
  );
};

export default memo(TokenSelector);
