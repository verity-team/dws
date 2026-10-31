"use client";

import { exchangeToReward, waitForDonation } from "@/utils/wallet/donate";
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
import { getUserDonationDataKeyFilter } from "@/api/dws/user/user";
import { useToggle } from "@/hooks/utils/useToggle";
import ThankDialog from "./ThankDialog";
import { TokenPrice } from "@/api/dws/donation/donation.type";
import { Maybe } from "@/utils";
import { Address } from "viem";

interface DonateFormProps {
  tokenPrices: TokenPrice[];
  rewardPrice: number;
  receiver: Maybe<Address>;
  // Why donating is not possible right now, null when it is
  blockedReason: Maybe<string>;
}

export interface DonateFormData {
  payAmount: number;
}

const DonateForm = ({
  tokenPrices,
  rewardPrice,
  receiver,
  blockedReason,
}: DonateFormProps): ReactElement<DonateFormProps> => {
  const userWallet = useContext(Wallet);
  const { requestTransaction } = useContext(WalletUtils);

  const [selectedToken, setSelectedToken] = useState<AvailableToken>("ETH");
  const [receiveAmount, setReceiveAmount] = useState<number | "N/A">(0);

  const [isLoading, setLoading] = useState(false);
  const [isConfirming, setConfirming] = useState(false);

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
    if (blockedReason != null || receiver == null) {
      toast.error(blockedReason ?? "Donating is not possible right now");
      return;
    }

    setLoading(true);

    if (!userWallet) {
      return;
    }

    try {
      const txHash = await requestTransaction(
        data.payAmount,
        selectedToken,
        receiver
      );
      if (!txHash) {
        setLoading(false);
        toast.error("Transaction rejected");
        return;
      }

      // Only a mined, successful transaction is a donation
      setConfirming(true);
      const toastId = toast.loading(
        "Transaction submitted, waiting for confirmation..."
      );
      const outcome = await waitForDonation(txHash);

      if (outcome === "reverted") {
        toast.error("Your transaction failed. No donation was made.", {
          id: toastId,
        });
        return;
      }

      if (outcome === "replaced") {
        toast.error(
          "Your transaction was cancelled or replaced in your wallet. No donation was made.",
          { id: toastId }
        );
        return;
      }

      if (outcome === "unknown") {
        toast.error(
          "Your transaction could not be confirmed yet. Please check your wallet and your donation history.",
          { id: toastId }
        );
        return;
      }

      toast.success("Thank you for your support!", { id: toastId });
      setThankOpen();

      // Revalidate user donations -- every page that is currently cached, the
      // new donation lands at the end of the history
      await mutate(getUserDonationDataKeyFilter(userWallet.wallet));

      const thankYouSection = document.getElementById("thank-you");
      if (!thankYouSection) {
        return;
      }
      thankYouSection.scrollIntoView({ behavior: "smooth" });
    } catch (error) {
      console.log(error);
    } finally {
      setConfirming(false);
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
        {blockedReason != null && (
          <div className="px-2">
            <TextError>{blockedReason}</TextError>
          </div>
        )}
        <div className="p-2">
          <ConnectButton
            disabled={
              receiveAmount === "N/A" ||
              receiveAmount <= 0 ||
              isNaN(rewardPrice) ||
              receiver == null ||
              blockedReason != null
            }
            account={userWallet.wallet}
            loading={isLoading}
            onDonate={handleSubmit(handleDonate)}
          />
          {isConfirming && (
            <div className="font-sans text-sm text-center mt-2">
              Transaction submitted, waiting for confirmation...
            </div>
          )}
          <div className="font-sans text-xs text-center mt-2">
            Smart-contract wallets (e.g. Safe or account-abstraction wallets)
            are not supported. Donations sent from them will not be credited.
          </div>
        </div>
      </form>
      <ThankDialog isOpen={isThankOpen} handleClose={setThankClose} />
    </>
  );
};

export default DonateForm;
