"use client";

import Image from "next/image";
import DonateForm from "./form/DonateForm";
import AFCForm from "./AFCForm";

const Donate = () => {
  return (
    <div className="flex flex-col transition-all">
      {/* Donate section */}
      <div className="max-w-2xl border-2 border-black rounded-2xl">
        <div className="w-full border-b-2 border-black bg-cred p-8 rounded-t-xl relative h-32">
          <Image
            src="/images/givememoney.jpeg"
            alt="shut up and take my money"
            fill
            className="rounded-t-xl object-cover"
          />
        </div>

        <div className="border-b-2 border-black">
          <div className="flex items-center justify-between p-4">
            <div>Current price: $0</div>
            <div>Next stage price: $0</div>
          </div>
        </div>

        <div className="bg-cblue py-2 rounded-b-2xl">
          <DonateForm />
        </div>
      </div>

      {/* Share section */}
      <div className="mt-8">
        <AFCForm />
      </div>
    </div>
  );
};

export default Donate;
