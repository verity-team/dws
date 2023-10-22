import { AvailableToken } from "@/utils/token";
import { ReactElement, memo } from "react";
import { UseFormRegister } from "react-hook-form";
import { DonateFormData } from "./DonateBox";

interface DonateInputProps {
  label: string;
  value: number | string;
  currency: AvailableToken | "MEMEME";
  register: UseFormRegister<DonateFormData>;
}

const DonateInput = ({
  label,
  value,
  currency,
}: DonateInputProps): ReactElement<DonateInputProps> => {
  return (
    <div className="rounded-lg bg-white border-2 border-black">
      <div className="px-4 py-2">{label}</div>
      <div className="px-4 py-2 flex items-center justify-between">
        <input
          className="bg-white border-none focus:border-none w-full"
          value={value}
        />
        <span>{currency}</span>
      </div>
    </div>
  );
};

export default memo(DonateInput);
