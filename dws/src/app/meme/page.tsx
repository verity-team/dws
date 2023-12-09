import MemeInput from "@/components/meme/MemeInput";

const MemePage = () => {
  return (
    <div className="grid grid-cols-12">
      <div className="col-span-3"></div>
      <div className="col-span-6">
        <h1 className="text-4xl font-bold">#Truthmemes</h1>
        <div className="mt-4">
          <MemeInput />
        </div>
      </div>
    </div>
  );
};

export default MemePage;
