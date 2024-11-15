import axios from 'axios';

export const BASE_URL: string = "http://a28e603c0d7c74a1fa16e29aa764dc90-236213353.ap-southeast-1.elb.amazonaws.com"
// export const BASE_URL: string = "http://0.0.0.0:5000";

export const axiosInstance = axios.create({
  baseURL: BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add a request interceptor
axiosInstance.interceptors.request.use(
  (config) => {
    const token = sessionStorage.getItem('access_token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);