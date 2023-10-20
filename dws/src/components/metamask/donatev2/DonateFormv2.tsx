"use client";

import { Nullable } from "@/utils/types";
import { useSDK } from "@metamask/sdk-react";
import Image from "next/image";
import { useCallback, useEffect, useMemo, useState } from "react";
import toast from "react-hot-toast";
import TokenSelector from "./TokenSelector";
import { AvailableToken, stableCoinPrice } from "@/utils/token";
import DonateInput from "./DonateInput";
import ConnectButton from "./ConnectBtn";
import { getUserDonationDataKey, useDonationData } from "@/utils/api/clientAPI";
import { donate, exchangeToReward } from "@/utils/metamask/donate";
import { mutate } from "swr";
import { useForm } from "react-hook-form";
import TextError from "@/components/common/TextError";

export interface DonateFormData {
  payAmount: number;
}

const DonateFormV2 = () => {
  const { sdk } = useSDK();
  const { tokenPrices } = useDonationData();

  const [account, setAccount] = useState<Nullable<string>>(null);
  const [selectedToken, setSelectedToken] = useState<AvailableToken>("ETH");
  const [receiveAmount, setReceiveAmount] = useState<number | "N/A">(0);

  const {
    register,
    handleSubmit,
    formState: { errors },
    getValues,
  } = useForm<DonateFormData>({
    defaultValues: { payAmount: 0 },
  });

  const handleTokenChange = useCallback((token: AvailableToken) => {
    setSelectedToken(token);
  }, []);

  const handleSwapTokenToReward = useCallback(() => {
    const [payAmount] = getValues(["payAmount"]);

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
      const reward = exchangeToReward(payAmount, selectedTokenPrice);
      setReceiveAmount(reward);
    } catch (err: any) {
      setReceiveAmount("N/A");
    }
  }, [getValues, selectedToken, tokenPrices]);

  useEffect(() => {
    handleSwapTokenToReward();
  }, [handleSwapTokenToReward]);

  const handleConnect = async (
    event: React.MouseEvent<HTMLButtonElement>
  ): Promise<void> => {
    event.preventDefault();

    try {
      if (sdk == null) {
        return;
      }
      const accounts = await sdk.connect();

      if (accounts == null || !Array.isArray(accounts)) {
        return;
      }
      setAccount(accounts[0]);
      toast("Welcome to TruthMemes", { icon: "👋" });
    } catch (err) {
      console.warn({ err });
    }
  };

  // If doing math, use toWei() + BN.js to avoid floating issues
  const handleDonate = async (data: DonateFormData) => {
    if (account == null) {
      return;
    }

    const txHash = await donate(account, data.payAmount, selectedToken);
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
    <div className="max-w-2xl border-2 border-black rounded-2xl">
      <div className="w-full border-b-2 border-black bg-cred p-8 rounded-t-xl relative h-32">
        {/* <div className="uppercase text-lg text-white text-center">donate ?</div> */}
        <Image
          src="/images/givememoney.jpeg"
          alt="shut up and take my money"
          fill={true}
          objectFit="cover"
          className="rounded-t-xl"
        />
      </div>

      <div className="border-b-2 border-black">
        <div className="flex items-center justify-between p-4">
          <div>Current price: 0</div>
          <div>Next stage price: 0</div>
        </div>
      </div>

      <div className="bg-cblue py-2 rounded-b-2xl">
        {/* <div className="grid grid-cols-3 gap-2 p-2">
          <button className="w-full py-2 rounded-2xl border-2 border-black bg-white">
            ETH
          </button>
          <button className="w-full py-2 rounded-2xl border-2 border-black bg-white">
            ETH
          </button>
          <button className="w-full py-2 rounded-2xl border-2 border-black bg-white">
            ETH
          </button>
        </div> */}

        <TokenSelector
          selectedToken={selectedToken}
          onTokenChange={handleTokenChange}
        />
        <form>
          <div className="grid grid-cols-2 gap-2 p-2">
            <div className="rounded-lg bg-white border-2 border-black">
              <div className="px-4 py-2">You pay</div>
              <div className="px-4 py-2 flex items-center justify-between">
                <input
                  className="bg-white border-none w-full focus:outline-none"
                  {...register("payAmount", {
                    valueAsNumber: true,
                    onChange: handleSwapTokenToReward,
                    required: true,
                    min: minDonateAmount,
                    validate: (value) => !isNaN(value),
                  })}
                />
                <span>{selectedToken}</span>
              </div>
              {/* {errors.payAmount && (
                <div className="mx-auto mt-2 w-11/12 font-sans">
                  {errors.payAmount.type === "required" && (
                    <TextError>The donate amount is required</TextError>
                  )}
                  {errors.payAmount.type === "min" && (
                    <TextError>
                      The minimum donate amount for {selectedToken} is{" "}
                      {minDonateAmount}
                    </TextError>
                  )}
                  {errors.payAmount.type === "validate" && (
                    <TextError>The donate amount should be a number</TextError>
                  )}
                </div>
              )} */}
            </div>
            <div className="rounded-lg bg-white border-2 border-black">
              <div className="px-4 py-2">You receive</div>
              <div className="px-4 py-2 flex items-center justify-between">
                <input
                  className="bg-white border-none focus:border-none w-full focus:outline-none"
                  disabled
                  value={receiveAmount}
                />
                <span>MEMEME</span>
              </div>
            </div>
          </div>
          <div className="p-2">
            <ConnectButton
              account={account}
              handleConnect={handleConnect}
              handleDonate={handleSubmit(handleDonate)}
            />
          </div>
        </form>
      </div>
    </div>
  );
};

export default DonateFormV2;
