import { Maybe } from "@/utils/types";
import { useSearchParams } from "next/navigation";

export const useAffiliateCode = (): Maybe<string> => {
  // Get affiliate code from URL
  const searchParams = useSearchParams();
  const affiliateCode = searchParams.get("afc");
  return affiliateCode;
};
