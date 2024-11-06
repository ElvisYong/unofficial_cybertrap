import { axiosInstance } from '@/data';
import { Template } from '@/app/types';

export const templateApi = {
  getAllTemplates: async (): Promise<Template[]> => {
    const response = await axiosInstance.get('/v1/templates');
    return response.data;
  }
}; 