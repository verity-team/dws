import MemeTimeline from "@/components/galactica/meme/MemeTimeline";

const TruthMemePage = () => {
  return (
    <div className="grid grid-cols-12 mt-12">
      <div className="col-span-12 md:col-span-6 md:col-start-3">
        <MemeTimeline />
      </div>
    </div>
  );
};

export default TruthMemePage;
