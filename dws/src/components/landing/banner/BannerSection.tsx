import { ReactElement, ReactNode } from "react";

interface BannerSectionProps {
  children: ReactNode;
  className: string | undefined;
}

const BannerSection = ({
  children,
  className,
}: BannerSectionProps): ReactElement<BannerSectionProps> => {
  return (
    <section className={className}>
      <div className="px-24 py-12 flex items-center justify-center">
        <div className="max-w-4xl flex flex-col items-start justify-center">
          {children}
        </div>
      </div>
    </section>
  );
};

export default BannerSection;
