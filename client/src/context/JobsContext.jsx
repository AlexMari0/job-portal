import { createContext, useContext, useEffect, useState } from "react";
import { useUser } from "@clerk/react";
import * as api from "../lib/api";

export const JobsContext = createContext();

export const JobsContextProvider = ({ children }) => {
  const { user } = useUser();

  const [jobs, setJobs] = useState([]);
  const [userApplications, setUserApplications] = useState([]);

  const fetchJobs = async () => {
    try {
      const data = await api.pub.get("/api/jobs");
      setJobs(data.jobs);
    } catch {}
  };

  const fetchUserApplications = async () => {
    try {
      const data = await api.user.get("/api/users/applications");
      setUserApplications(data.applications);
    } catch {}
  };

  useEffect(() => {
    fetchJobs();
  }, []);

  useEffect(() => {
    if (user) {
      fetchUserApplications();
    }
  }, [user]);

  const value = {
    jobs,
    setJobs,
    userApplications,
    setUserApplications,
    fetchUserApplications,
  };

  return <JobsContext.Provider value={value}>{children}</JobsContext.Provider>;
};
