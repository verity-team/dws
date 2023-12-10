import ClientRoot from "@/components/ClientRoot";
import SignInBtn from "@/components/galactica/account/SignInBtn";
import MemeInput from "@/components/galactica/meme/MemeInput";

const MemePage = () => {
  return (
    <div className="grid grid-cols-12">
      <div className="col-span-3"></div>
      <div className="col-span-6">
        <ClientRoot>
          <div className="flex items-center justify-between">
            <h1 className="text-4xl font-bold">#Truthmemes</h1>
            <SignInBtn />
          </div>
          <div className="mt-4">
            <MemeInput />
          </div>
        </ClientRoot>
      </div>
      <div className="col-span-3"></div>
    </div>
  );
};

export default MemePage;
