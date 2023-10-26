"use client";

import { subscribeEmail } from "@/utils/api/client/emailAPI";
import { FormEvent, useRef, useState } from "react";
import toast from "react-hot-toast";

const Newsletter = () => {
  const [subscribed, setSubscribed] = useState(false);
  const emailRef = useRef<HTMLInputElement>(null);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    const email = emailRef.current?.value;
    if (email == null || email.trim() === "") {
      toast.error("Please input email before subscribe");
      return;
    }

    try {
      await subscribeEmail(email);
      setSubscribed(true);
    } catch (error: any) {
      toast.error(error.message);
    }
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
          {subscribed ? (
            <div className="text-2xl font-sans">
              Thank you for your subscription
            </div>
          ) : (
            <form onSubmit={handleSubmit} className="w-full">
              <input
                className="p-4 w-1/3 border-2 border-gray-400"
                placeholder="Enter your email"
                required
                type="email"
                ref={emailRef}
              />
              <button
                type="submit"
                className="p-4 w-1/3 uppercase bg-cblack text-white text-xl"
              >
                Submit
              </button>
            </form>
          )}
        </div>
      </div>
    </div>
  );
};

export default Newsletter;
