import { MemeFilter } from "../../meme/meme.type";
import MemeTimeline from "../MemeTimeline";
import MemeItem from "./MemeItem";

const USER_MEME_FILTER: MemeFilter = {
  status: "APPROVED",
};

const UserMemeTimeline = () => {
  return (
    <div>
      <MemeTimeline filter={USER_MEME_FILTER} ItemLayout={MemeItem} />
    </div>
  );
};

export default UserMemeTimeline;
