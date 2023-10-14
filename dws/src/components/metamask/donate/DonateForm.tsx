import { ReactElement, useCallback, useEffect, useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import TextError from "@/components/common/TextError";
import { donate, exchangeToReward } from "@/utils/metamask/donate";
import { AvailableToken, stableCoinPrice } from "@/utils/token";
import { mutate } from "swr";
import { getUserDonationDataKey, useDonationData } from "@/utils/api/clientAPI";

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
  const [rewardToken, setRewardToken] = useState<number | "N/A">(0);

  const { status: campaignStatus } = useDonationData();

  const {
    register,
    handleSubmit,
    formState: { errors },
    getValues,
  } = useForm<DonateFormData>();

  const { tokenPrices } = useDonationData();

  const handleSwapTokenToReward = useCallback(() => {
    const [amount] = getValues(["amount"]);

    // TODO: Use API to get token price
    let selectedTokenPrice = Number(
      tokenPrices.find((price) => price.asset.toUpperCase() === selectedToken)
        ?.price
    );

    // Try to get price for stable coin if API prices failed
    if (isNaN(selectedTokenPrice)) {
      selectedTokenPrice = Number(stableCoinPrice[selectedToken]);
    }

    try {
      const reward = exchangeToReward(amount, selectedTokenPrice);
      setRewardToken(reward);
    } catch (err: any) {
      setRewardToken("N/A");
    }
  }, [getValues, selectedToken, tokenPrices]);

  useEffect(() => {
    handleSwapTokenToReward();
  }, [handleSwapTokenToReward]);

  // If doing math, use toWei() + BN.js to avoid floating issues
  const handleDonate = async (data: DonateFormData) => {
    const txHash = await donate(account, data.amount, selectedToken);
    if (txHash == null) {
      alert("Failed to donate token.");
      return;
    }

    // TODO: Change this to Toast for better UX
    alert("Transaction sent");

    // Revalidate user donations
    await mutate(getUserDonationDataKey(account));

    return txHash;
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
          <label className="my-1" htmlFor="donate-amount">
            Amount*
          </label>
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
              {selectedToken} for {rewardToken} token(s)
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
      <button
        type="submit"
        className="px-4 py-2 rounded-lg border-2 mt-2 border-gray-700 disabled:border-gray-500 disabled:text-gray-500 disabled:cursor-not-allowed"
        disabled={campaignStatus !== "open"}
      >
        Donate
      </button>
    </form>
  );
};

export default DonateForm;
