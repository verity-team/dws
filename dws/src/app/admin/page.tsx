"use client";

import AdminTimeline from "@/components/admin/AdminTimeline";
import { MemeUploadStatus } from "@/components/galactica/meme/meme.type";
import { Box, Tabs, Tab } from "@mui/material";
import { SyntheticEvent, useState } from "react";

const categories: MemeUploadStatus[] = ["PENDING", "APPROVED", "DENIED"];

const AdminPage = () => {
  const [selectedCategory, setSelectedCategory] =
    useState<MemeUploadStatus>("PENDING");

  const handleCategoryChange = (
    event: SyntheticEvent,
    value: MemeUploadStatus
  ) => {
    setSelectedCategory(value);
  };

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
