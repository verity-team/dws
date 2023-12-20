import MemeTimeline from "@/components/galactica/meme/MemeTimeline";
import { MemeUploadStatus } from "@/components/galactica/meme/meme.type";

const categories: MemeUploadStatus[] = ["PENDING", "APPROVED", "DENIED"];

const AdminPage = () => {
  return (
    <div className="grid grid-cols-12">
      <div className="col-span-3"></div>
      <div className="col-span-6">
        <MemeTimeline filter={{ status: "PENDING" }} />
      </div>
      <div className="col-span-3"></div>
    </div>
  );
};

export default AdminPage;
