import ClientRoot from "@/components/ClientRoot";
import SignInBtn from "@/components/galactica/account/SignInBtn";
import MemeInput from "@/components/galactica/meme/MemeInput";
import MemeList from "@/components/galactica/meme/list/MemeList";

const MemePage = () => {
  return (
    <div className="grid grid-cols-12">
      <div className="col-span-2"></div>
      <div className="col-span-8">
        <ClientRoot>
          <div className="flex items-center justify-between">
            <h1 className="text-4xl font-bold">#Truthmemes</h1>
            <SignInBtn />
          </div>
          <div className="mt-4">
            <MemeInput />
          </div>
          <div>
            <MemeList />
          </div>
        </ClientRoot>
      </div>
      <div className="col-span-2"></div>
    </div>
  );
};

export default MemePage;
