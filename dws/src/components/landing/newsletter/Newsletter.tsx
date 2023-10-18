"use client";

import toast from "react-hot-toast";

const Newsletter = () => {
  const handleSubmit = () => {
    toast("Email subscription will be available soon", {
      icon: "⚠️",
      style: {
        fontSize: "1rem",
      },
    });
  };

  return (
    <div className="bg-cblue">
      <div className="mx-8 p-20">
        <div>
          <h3 className="text-3xl leading-loose">
            Get notified. Get <span className="text-cred">Memes</span>.
          </h3>
          <p className="text-base font-sans font-bold">
            No SPAM, just News and Memes. Unsubscribe any time.
          </p>
        </div>
        <div className="mt-12 space-x-4 flex items-center">
          <input
            className="p-4 w-1/3 border-2 border-gray-400"
            placeholder="Enter your email"
            required
          />
          <button
            type="button"
            className="p-4 w-1/3 uppercase bg-cblack text-white text-xl"
            onClick={handleSubmit}
          >
            Submit
          </button>
        </div>
      </div>
    </div>
  );
};

export default Newsletter;
