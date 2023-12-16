import MemePage from "@/components/galactica/meme/MemePage";

const TruthMemePage = () => {
  return (
    <div className="grid grid-cols-12">
      <div className="col-span-3"></div>
      <div className="col-span-6">
        <MemePage />
      </div>
      <div className="col-span-3"></div>
    </div>
  );
};

export default TruthMemePage;
