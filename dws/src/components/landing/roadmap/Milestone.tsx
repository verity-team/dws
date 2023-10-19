import { ReactElement, memo } from "react";

interface MilestoneProps {
  text: string;
  time: string;
}

const Milestone = ({
  text,
  time,
}: MilestoneProps): ReactElement<MilestoneProps> => {
  return (
    <div className="col-span-1 place-self-center relative">
      <div className="flex flex-col items-center justify-center">
        {/* <div className="p-2 rounded-full bg-black w-4 h-4"></div> */}
        <div className="text-3xl">{text}</div>
        <div className="text-base leading-5 mt-1 font-sans opacity-70">
          {time}
        </div>
      </div>
    </div>
  );
};

export default memo(Milestone);
