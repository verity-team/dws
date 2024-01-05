import clsx from "clsx";
import { HTMLAttributes, ReactElement, ReactNode } from "react";

interface BannerSectionProps {
  children: ReactNode;
  className?: HTMLAttributes<HTMLDivElement>["className"];
}

const BannerSection = ({
  children,
  className,
}: BannerSectionProps): ReactElement<BannerSectionProps> => {
  return (
    <section className={clsx(className, "mx-8")}>
      <div className="max-w-4xl flex flex-col items-start justify-center">
        {children}
      </div>
    </section>
  );
};

export default BannerSection;
