import { axiosInstance } from '@/data';
import { Domain } from '@/app/types';

export const domainApi = {
  getAllDomains: async (): Promise<Domain[]> => {
    const response = await axiosInstance.get('/v1/domains');
    return response.data;
  },

  addDomain: async (domain: string): Promise<Domain> => {
    const response = await axiosInstance.post('/v1/domains', { domain });
    return response.data;
  },

  deleteDomain: async (id: string): Promise<void> => {
    await axiosInstance.delete(`/v1/domains/${id}`);
  }
}; 