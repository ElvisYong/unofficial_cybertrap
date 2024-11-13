import { axiosInstance } from '@/data';
import { Scan, ScheduledScanResponse } from '@/app/types';

export const scanApi = {
  getScheduledScans: async (): Promise<ScheduledScanResponse[]> => {
    const response = await axiosInstance.get('/v1/scans/scheduled');
    return response.data;
  },

  scheduleScan: async (domainId: string, templateIds: string[], scheduledDate: string): Promise<ScheduledScanResponse> => {
    const response = await axiosInstance.post('/v1/scans/schedule', {
      domainId,
      templateIds,
      scheduledDate
    });
    return response.data;
  },

  scanAll: async (domains: string[]): Promise<void> => {
    await axiosInstance.post('/v1/scans/all', { domains });
  },

  getAllScans: async (): Promise<Scan[]> => {
    const response = await axiosInstance.get('/v1/scans');
    return response.data;
  },

  getMultiScans: async (): Promise<Scan[]> => {
    const response = await axiosInstance.get('/v1/scans/multi');
    return response.data;
  },

  scanDomains: async (domainIds: string[], templateIds: string[], scanAllNuclei: boolean = false): Promise<void> => {
    await axiosInstance.post('/v1/scans', {
      domainIds,
      templateIds,
      scanAllNuclei
    });
  },

  deleteScheduledScan: async (id: string): Promise<void> => {
    await axiosInstance.delete(`/v1/scans/schedule?id=${id}`);
  }
}; 