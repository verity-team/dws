"use client";

import { exchangeToReward } from "@/utils/wallet/donate";
import ConnectButton from "./ConnectBtn";
import TokenSelector from "./TokenSelector";
import { AvailableToken, stableCoinPrice } from "@/utils/wallet/token";
import {
  useState,
  useCallback,
  useEffect,
  useMemo,
  ReactElement,
  useContext,
} from "react";
import { useForm } from "react-hook-form";
import toast from "react-hot-toast";
import { mutate } from "swr";
import TextError from "@/components/common/TextError";
import { Wallet, WalletUtils } from "@/components/ClientRoot";
import { getUserDonationDataKey } from "@/api/dws/user/user";
import { useToggle } from "@/hooks/utils/useToggle";
import ThankDialog from "./ThankDialog";
import { TokenPrice } from "@/api/dws/donation/donation.type";

interface DonateFormProps {
  tokenPrices: TokenPrice[];
  rewardPrice: number;
}

export interface DonateFormData {
  payAmount: number;
}

const DonateForm = ({
  tokenPrices,
  rewardPrice,
}: DonateFormProps): ReactElement<DonateFormProps> => {
  const userWallet = useContext(Wallet);
  const { requestTransaction } = useContext(WalletUtils);

  const [selectedToken, setSelectedToken] = useState<AvailableToken>("ETH");
  const [receiveAmount, setReceiveAmount] = useState<number | "N/A">(0);

  const [isLoading, setLoading] = useState(false);

  const {
    isOpen: isThankOpen,
    open: setThankOpen,
    close: setThankClose,
  } = useToggle();

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

    let selectedTokenPrice = Number(
      tokenPrices.find((price) => price.asset.toUpperCase() === selectedToken)
        ?.price
    );

    // Try to get price for stable coin if API prices failed
    if (isNaN(selectedTokenPrice)) {
      selectedTokenPrice = Number(stableCoinPrice[selectedToken]);
    }

    try {
      const reward = exchangeToReward(
        payAmount,
        selectedTokenPrice,
        rewardPrice
      );
      setReceiveAmount(reward);
    } catch (err: any) {
      if (payAmount === 0) {
        setReceiveAmount(0);
        return;
      }
      setReceiveAmount("N/A");
    }
  }, [getValues, selectedToken, tokenPrices, rewardPrice]);

  // Update price when
  useEffect(() => {
    handleSwapTokenToReward();
  }, [selectedToken]);

  const handleDonate = async (data: DonateFormData) => {
    setLoading(true);

    if (!userWallet) {
      return;
    }

    try {
      const txHash = await requestTransaction(data.payAmount, selectedToken);
      if (!txHash) {
        setLoading(false);
        toast.error("Transaction rejected");
        return;
      }

      toast.success("Thank you for your support!");
      setThankOpen();

      // Revalidate user donations
      await mutate(getUserDonationDataKey(userWallet.wallet));

      const thankYouSection = document.getElementById("thank-you");
      if (!thankYouSection) {
        return;
      }
      thankYouSection.scrollIntoView({ behavior: "smooth" });
    } catch (error) {
      console.log(error);
    } finally {
      setLoading(false);
    }
  };

  const minDonateAmount: number = useMemo(() => {
    if (selectedToken === "ETH") {
      const minETH = Number(process.env.NEXT_PUBLIC_MIN_ETH);
      if (isNaN(minETH)) {
        return 0.005;
      }
      return minETH;
    }

    if (selectedToken === "USDT" || selectedToken === "USDC") {
      const minUSDT = Number(process.env.NEXT_PUBLIC_MIN_USDT);
      if (isNaN(minUSDT)) {
        return 5;
      }
      return minUSDT;
    }

    return 0;
  }, [selectedToken]);

  return (
    <>
      <div className="px-3">Pay with</div>
      <TokenSelector
        selectedToken={selectedToken}
        onTokenChange={handleTokenChange}
      />
      <div className="px-3 mt-2">Payment info</div>
      <form>
        <div className="grid grid-cols-1 gap-2 p-2 md:grid-cols-2">
          <div className="rounded-lg bg-white border-2 border-black">
            <div className="px-4 py-2">You pay</div>
            <div className="px-4 py-2 flex items-center justify-between">
              <input
                className="bg-white border-none w-full focus:outline-none"
                autoComplete="off"
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
          </div>
          <div className="rounded-lg bg-white border-2 border-black">
            <div className="px-4 py-2">You receive</div>
            <div className="px-4 py-2 flex items-center justify-between">
              <input
                className="bg-white border-none focus:border-none w-full focus:outline-none"
                disabled
                value={receiveAmount}
                autoComplete="off"
              />
              <span>$TRUTH</span>
            </div>
          </div>
        </div>
        {errors.payAmount && (
          <div className="p-2">
            {errors.payAmount.type === "required" && (
              <TextError>Buy amount is required</TextError>
            )}
            {errors.payAmount.type === "min" && (
              <TextError>
                The minimum buy amount for {selectedToken} is {minDonateAmount}
              </TextError>
            )}
            {errors.payAmount.type === "validate" && (
              <TextError>Buy amount should be a number</TextError>
            )}
          </div>
        )}
        <div className="p-2">
          <ConnectButton
            disabled={
              receiveAmount === "N/A" ||
              receiveAmount <= 0 ||
              isNaN(rewardPrice)
            }
            account={userWallet.wallet}
            loading={isLoading}
            onDonate={handleSubmit(handleDonate)}
          />
        </div>
      </form>
      <ThankDialog isOpen={isThankOpen} handleClose={setThankClose} />
    </>
  );
};

export default DonateForm;
