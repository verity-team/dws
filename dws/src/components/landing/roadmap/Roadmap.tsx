import Image from "next/image";
import Milestone from "./Milestone";

const Roadmap = () => {
  return (
    <div className="mx-6 my-10">
      <div className="flex flex-col items-center justify-center">
        {/* Banner */}
        <Image
          src="/dws-images/roadmap.png"
          alt="roadmap"
          width={500}
          height={0}
          className="h-auto"
        />
        <p className="text-xl">
          If you want to go fast, go alone, if you want to go far, go together.
        </p>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-4 my-16 space-y-12 md:space-y-0">
        <Milestone text="Presale" time="October 31, 2023" />
        <Milestone text="Token" time="January 3, 2024" />
        <Milestone text="Launch v1" time="April 20, 2024" />
        <Milestone text="Decentralize" time="2025" />
      </div>
    </div>
  );
};

export default Roadmap;
