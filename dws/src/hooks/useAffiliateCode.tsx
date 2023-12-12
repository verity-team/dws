import { Maybe } from "@/utils";
import { useSearchParams } from "next/navigation";

export const useAffiliateCode = (): Maybe<string> => {
  // Get affiliate code from URL
  const searchParams = useSearchParams();
  const affiliateCode = searchParams.get("afc");
  return affiliateCode;
};
