"use client";

import ClientRoot from "@/components/ClientRoot";
import MemeNavbar from "@/components/galactica/meme/common/MemeNavbar";
import UserMemeTimeline from "@/components/galactica/memeV2/user/UserMemeTimeline";

const TruthMemePage = () => {
  return (
    <ClientRoot>
      <div className="grid grid-cols-12 mt-12 mx-auto">
        <div className="col-span-12">
          <MemeNavbar />
        </div>
        <div className="col-span-12 md:col-span-4 md:col-start-5">
          <UserMemeTimeline />
        </div>
      </div>
    </ClientRoot>
  );
};

export default TruthMemePage;
