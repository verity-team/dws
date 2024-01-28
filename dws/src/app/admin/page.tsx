"use client";

import {
  requestAccessTokenVerification,
  requestSignInWithCredentials,
} from "@/api/galactica/admin/admin";
import AdminTimeline from "@/components/admin/AdminTimeline";
import TextError from "@/components/common/TextError";
import { MemeUploadStatus } from "@/components/galactica/meme/meme.type";
import useAccountId from "@/hooks/store/useAccountId";
import { Box, Tabs, Tab } from "@mui/material";
import Image from "next/image";
import { SyntheticEvent, useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import toast from "react-hot-toast";

const categories: MemeUploadStatus[] = ["PENDING", "APPROVED", "DENIED"];

interface AdminSignInFormData {
  username: string;
  password: string;
}

const AdminPage = () => {
  const [selectedCategory, setSelectedCategory] =
    useState<MemeUploadStatus>("PENDING");

  const [signedIn, setSignedIn] = useState(false);
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<AdminSignInFormData>({
    defaultValues: {
      username: "",
      password: "",
    },
  });

  const { accessToken, setAccessToken } = useAccountId();

  useEffect(() => {
    if (!accessToken) {
      setSignedIn(false);
      return;
    }

    requestAccessTokenVerification(accessToken).then((isAuthenticated) => {
      if (!isAuthenticated) {
        setSignedIn(false);
        return;
      }

      // Usable access token
      setSignedIn(true);
    });
  }, [accessToken]);

  const handleCategoryChange = (_: SyntheticEvent, value: MemeUploadStatus) => {
    setSelectedCategory(value);
  };

  const handleSignInWithCredentials = async (data: AdminSignInFormData) => {
    const accessToken = await requestSignInWithCredentials(
      data.username,
      data.password
    );
    if (accessToken == null) {
      toast.error("Wrong username or password!");
      return;
    }
    setAccessToken(accessToken);
    setSignedIn(true);
  };

  if (!signedIn) {
    return (
      <div className="container mx-auto grid grid-cols-2">
        <div className="mx-auto w-3/4">
          <div className="text-4xl">Truth Memes Portal</div>
          <form
            onSubmit={handleSubmit(handleSignInWithCredentials)}
            className="mt-12 space-y-4"
          >
            <div className="flex flex-col items-start space-y-2 w-full">
              <label htmlFor="username" className="text-xl">
                Username
              </label>
              <input
                id="username"
                className="text-lg px-4 py-2 border border-slate-200 rounded-lg w-full"
                placeholder="username"
                autoComplete="username"
                {...register("username", {
                  required: "This field is required",
                })}
              />
              {errors.username?.type === "required" && (
                <TextError>{errors.username.message}</TextError>
              )}
            </div>
            <div className="flex flex-col items-start space-y-2">
              <label htmlFor="password">Password</label>
              <input
                id="password"
                type="password"
                className="text-lg px-4 py-2 border border-slate-200 rounded-lg w-full"
                placeholder="password"
                autoComplete="current-password"
                {...register("password", {
                  required: "This field is required",
                })}
              />
              {errors.password?.type === "required" && (
                <TextError>{errors.password.message}</TextError>
              )}
            </div>
            <button
              type="submit"
              className="text-lg px-4 py-2 bg-blue-500 hover:bg-blue-600 rounded-lg text-white"
            >
              Sign in
            </button>
          </form>
        </div>
        <div className="relative mt-12 flex items-center justify-center">
          <Image
            src="/dws-images/logo.png"
            width={200}
            height={200}
            alt="eye of truth"
            className="w-auto h-auto"
            priority={true}
          />
        </div>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-12 relative">
      <div className="col-span-3"></div>
      <div className="col-span-6">
        <Box sx={{ width: "100%" }}>
          <Box
            sx={{
              borderBottom: 1,
              borderColor: "divider",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              position: "sticky",
            }}
          >
            <Tabs
              value={selectedCategory}
              onChange={handleCategoryChange}
              aria-label="basic tabs example"
            >
              {categories.map((category) => (
                <Tab key={category} label={category} value={category} />
              ))}
            </Tabs>
          </Box>
        </Box>
        <AdminTimeline filter={{ status: selectedCategory }} />
      </div>
      <div className="col-span-3"></div>
    </div>
  );
};

export default AdminPage;
