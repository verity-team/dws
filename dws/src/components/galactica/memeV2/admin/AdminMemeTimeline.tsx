import { MemeFilter } from "../../meme/meme.type";
import MemeTimeline from "../MemeTimeline";
import MemeItem from "./MemeItem";

interface AdminMemeTimelineProps {
  filter: MemeFilter;
}

const AdminMemeTimeline = ({ filter }: AdminMemeTimelineProps) => {
  // Default for admin to see pending memes on page load
  return (
    <div className="w-full">
      <MemeTimeline filter={filter} ItemLayout={MemeItem} />
    </div>
  );
};

export default AdminMemeTimeline;
