import { ReactElement } from "react";
import { MemeFilter } from "../meme/meme.type";
import { MemeUpload } from "@/api/galactica/meme/meme.type";

interface MemeTimelineProps {
  filter: MemeFilter;
  itemLayout: ReactElement<MemeUpload>;
}

const MemeTimeline =
  ({}: MemeTimelineProps): ReactElement<MemeTimelineProps> => {
    return <div></div>;
  };

export default MemeTimeline;
