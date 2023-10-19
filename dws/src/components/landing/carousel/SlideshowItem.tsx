import { CSSProperties, ReactElement, ReactNode } from "react";

interface SlideshowItemProps {
  children: ReactNode;
  isActive: boolean;
  currentItem: number;
}

const SlideshowItem = ({
  children,
  isActive,
  currentItem,
}: SlideshowItemProps): ReactElement<SlideshowItemProps> => {
  const overlayEffect: CSSProperties = isActive
    ? {
        opacity: 1,
        filter: "blur(0px)",
        transform: `translate3d(${
          -400 * currentItem
        }px, 0px, 0px) scale3d(1, 1, 1) rotateX(0deg) rotateY(0deg) rotateZ(0deg) skew(0deg, 0deg)`,
      }
    : {
        opacity: 0.5,
        filter: "blur(5px)",
        transform: `translate3d(${
          -400 * currentItem
        }px, 0px, 0px) scale3d(0.8, 0.8, 1) rotateX(0deg) rotateY(0deg) rotateZ(0deg) skew(0deg, 0deg)`,
      };

  return (
    <div
      style={{
        ...overlayEffect,
        transformStyle: "preserve-3d",
        transition: "transform 500ms ease 0s",
      }}
      className="relative inline-block w-full h-full overflow-hidden"
    >
      <div className="flex items-center justify-center w-full h-full">
        {children}
      </div>
    </div>
  );
};

export default SlideshowItem;
