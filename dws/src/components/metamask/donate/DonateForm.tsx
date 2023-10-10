import { ReactElement, useCallback, useEffect, useMemo, useState } from "react";
import { Validate, ValidateResult, useForm } from "react-hook-form";
import { Undefinable } from "@/utils/types";
import { toWei } from "web3-utils";
import { AvailableToken } from "./Donate";
import TextError from "@/components/common/TextError";

export interface DonateFormData {
  amount: number;
}

interface DonateFormProps {
  selectedToken: AvailableToken;
  account: string;
}

const DonateForm = ({
  selectedToken,
  account,
}: DonateFormProps): ReactElement<DonateFormData> => {
  const [rewardToken, setRewardToken] = useState<number>(0);

  const {
    register,
    handleSubmit,
    formState: { errors },
    getValues,
  } = useForm<DonateFormData>();

  // TODO: Connect this with price API later
  // TODO: Add BN.js for all calculation to avoid floating point errors
  const handleSwapTokenToReward = useCallback(() => {
    const [amount] = getValues(["amount"]);

    const rewardTokenPrice = process.env
      .NEXT_PUBLIC_REWARD_PRICE as Undefinable<number>;
    if (!rewardTokenPrice) {
      // Abort operation if cannot get reward price
      alert("Cannot get the reward price at the moment!");
      return;
    }

    const tokenPrice = selectedToken === "ETH" ? 1600 : 1;
    const reward = Math.ceil((amount * tokenPrice) / rewardTokenPrice);

    setRewardToken(isNaN(reward) ? 0 : reward);
  }, [getValues, selectedToken]);

  useEffect(() => {
    handleSwapTokenToReward();
  }, [handleSwapTokenToReward]);

  // If doing math, use toWei() + BN.js to avoid floating issues
  const handleDonate = async (data: DonateFormData) => {
    const ethereum = window.ethereum;
    if (ethereum == null) {
      alert("Ethereum object not available in window. Check your Metamask");
      return;
    }

    const receiveWallet = process.env.NEXT_PUBLIC_DONATE_PUBKEY;
    if (receiveWallet == null) {
      alert("Receive wallet not set. Unable to donate");
      return;
    }

    try {
      const txHash = await ethereum.request({
        method: "eth_sendTransaction",
        params: [
          {
            from: account,
            to: receiveWallet,
            value: toWei(data.amount, "ether"),
            // gasLimit: '0x5028', // Customizable by the user during MetaMask confirmation.
            // maxPriorityFeePerGas: '0x3b9aca00', // Customizable by the user during MetaMask confirmation.
            // maxFeePerGas: '0x2540be400', // Customizable by the user during MetaMask confirmation.
          },
        ],
      });

      alert(`Successfully transfered. TxHash: ${txHash}`);
    } catch (err: any) {
      console.log(err.message);
    }
  };

  // TODO: Change fallback value if needed
  const minDonateAmount: number = useMemo(() => {
    if (selectedToken === "ETH") {
      const minETH = Number(process.env.NEXT_PUBLIC_MIN_ETH);
      if (isNaN(minETH)) {
        return 0.005;
      }
      return minETH;
    }

    if (selectedToken === "USDT") {
      const minUSDT = Number(process.env.NEXT_PUBLIC_MIN_USDT);
      if (isNaN(minUSDT)) {
        return 5;
      }
      return minUSDT;
    }

    return 0;
  }, [selectedToken]);

  return (
    <form onSubmit={handleSubmit(handleDonate)}>
      <div>
        <div className="flex flex-col mt-2">
          <label htmlFor="donate-amount">Amount*</label>
          <div>
            <input
              className="w-1/5 px-4 py-2 rounded-lg border focus:border-2"
              placeholder="0"
              id="donate-amount"
              {...register("amount", {
                valueAsNumber: true,
                onChange: handleSwapTokenToReward,
                required: true,
                min: minDonateAmount,
                validate: (value) => !isNaN(value),
              })}
            />
            <span className="px-2">
              {selectedToken} for {rewardToken} GMS Token
            </span>
          </div>
        </div>
        {errors.amount && (
          <div className="mt-2">
            {errors.amount.type === "required" && (
              <TextError>The donate amount is required</TextError>
            )}
            {errors.amount.type === "min" && (
              <TextError>
                The minimum donate amount for {selectedToken} is{" "}
                {minDonateAmount}
              </TextError>
            )}
            {errors.amount.type === "validate" && (
              <TextError>The donate amount should be a number</TextError>
            )}
          </div>
        )}
      </div>
      <button type="submit" className="px-4 py-2 rounded-lg border-2 mt-2">
        Donate
      </button>
    </form>
  );
};

export default DonateForm;
