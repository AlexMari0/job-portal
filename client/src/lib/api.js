import axios from "axios";
import { toast } from "react-toastify";

const backendUrl = import.meta.env.VITE_BACKEND_URL;

let companyToken = null;
let clerkGetToken = null;

export function setCompanyToken(token) {
  companyToken = token;
}

export function setClerkGetToken(fn) {
  clerkGetToken = fn;
}

async function request(method, path, { data, headers = {} } = {}) {
  try {
    const res = await axios({ method, url: backendUrl + path, data, headers });
    if (res.data.success) {
      return res.data;
    }
    toast.error(res.data.message);
    throw new Error(res.data.message);
  } catch (error) {
    if (!error.message || error.response) {
      toast.error(error.response?.data?.message || error.message);
    }
    throw error;
  }
}

function companyHeaders() {
  return { token: companyToken };
}

async function userHeaders() {
  const token = clerkGetToken ? await clerkGetToken() : null;
  return { Authorization: `Bearer ${token}` };
}

export const company = {
  get: (path) => request("get", path, { headers: companyHeaders() }),
  post: (path, data) => request("post", path, { data, headers: companyHeaders() }),
};

export const user = {
  get: async (path) => request("get", path, { headers: await userHeaders() }),
  post: async (path, data) => request("post", path, { data, headers: await userHeaders() }),
};

export const pub = {
  get: (path) => request("get", path),
  post: (path, data) => request("post", path, { data }),
};
