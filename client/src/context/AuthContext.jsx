import { createContext, useEffect, useState } from "react";
import { useAuth, useUser } from "@clerk/react";
import * as api from "../lib/api";

export const AuthContext = createContext();

export const AuthContextProvider = ({ children }) => {
  const { user } = useUser();
  const { getToken } = useAuth();

  const [showRecruiterLogin, setShowRecruiterLogin] = useState(false);
  const [companyToken, setCompanyToken] = useState(null);
  const [companyData, setCompanyData] = useState(null);
  const [userData, setUserData] = useState(null);

  const fetchCompanyData = async () => {
    try {
      const data = await api.company.get("/api/company/company");
      setCompanyData(data.company);
    } catch {}
  };

  const fetchUserData = async () => {
    try {
      const data = await api.user.get("/api/users/user");
      setUserData(data.user);
    } catch {}
  };

  const loginCompany = (token, company) => {
    setCompanyToken(token);
    setCompanyData(company);
    localStorage.setItem("companyToken", token);
  };

  const logoutCompany = () => {
    setCompanyToken(null);
    setCompanyData(null);
    localStorage.removeItem("companyToken");
  };

  useEffect(() => {
    const stored = localStorage.getItem("companyToken");
    if (stored) {
      setCompanyToken(stored);
    }
  }, []);

  useEffect(() => {
    api.setCompanyToken(companyToken);
    if (companyToken) {
      fetchCompanyData();
    }
  }, [companyToken]);

  useEffect(() => {
    api.setClerkGetToken(getToken);
  }, [getToken]);

  useEffect(() => {
    if (user) {
      fetchUserData();
    }
  }, [user]);

  const value = {
    showRecruiterLogin,
    setShowRecruiterLogin,
    companyToken,
    companyData,
    userData,
    setUserData,
    fetchUserData,
    loginCompany,
    logoutCompany,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};
