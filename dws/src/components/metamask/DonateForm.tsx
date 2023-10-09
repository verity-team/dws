import { useCallback, useState } from "react";
import { useForm } from "react-hook-form";
import { Undefinable } from "@/utils/types";

interface DonateFormData {
  amount: number;
  rewardAmount: number;
}

type AvailableToken = "ETH" | "USDT";

const avaiableTokens: Array<AvailableToken> = ["ETH", "USDT"];

const DonateForm = () => {
  const [selectedToken, setSelectedToken] = useState<AvailableToken>("ETH");
  const [rewardToken, setRewardToken] = useState<number>(0);

  const {
    register,
    handleSubmit,
    formState: { errors },
    getValues,
  } = useForm<DonateFormData>();

  const handleChangeToken = useCallback((token: AvailableToken): void => {
    setSelectedToken(token);
  }, []);

  // TODO: Connect this with price API later
  const handleSwapTokenToReward = () => {
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
  };

  const handleDonation = () => {};

  return (
    <form onSubmit={handleSubmit(handleDonation)}>
      <div className="flex flex-col">
        <div className="py-2">Donate in</div>
        {avaiableTokens.map((token) => (
          <div key={token} className="py-1">
            <input
              type="radio"
              name="token"
              value={token}
              checked={selectedToken === token}
              onChange={() => handleChangeToken(token)}
            />
            <label>{token}</label>
          </div>
        ))}
      </div>

      <div className="flex px-4 py-2 justify-start items-center"></div>
      <div>
        <input
          className="w-1/5 px-4 py-2 rounded-lg border focus:border-2"
          defaultValue={0}
          {...register("amount", {
            required: true,
            valueAsNumber: true,
            onChange: handleSwapTokenToReward,
          })}
        />
        <span className="px-1">{selectedToken}</span>
        <div className="flex justify-start items-center m-2">FOR</div>
        <span className="px-1"> {rewardToken} GMS Token</span>
      </div>

      <button type="submit" className="px-4 py-2 rounded-lg border-2 mt-3">
        Donate
      </button>
    </form>
  );
};

export default DonateForm;
