import { axiosInstance } from '@/data';
import { Scan, ScheduledScanResponse } from '@/app/types';

export const scanApi = {
  getScheduledScans: async (): Promise<ScheduledScanResponse[]> => {
    try {
      const response = await axiosInstance.get('/v1/scans/schedule');
      return response.data;
    } catch (error) {
      console.error('Error fetching scheduled scans:', error);
      throw new Error('Failed to fetch scheduled scans.');
    }
  },

  scheduleScan: async (domainId: string, templateIds: string[], scheduledDate: string, scanAll: boolean = false): Promise<ScheduledScanResponse> => {
    try {
      const response = await axiosInstance.post('/v1/scans/schedule', {
        domainIds: [domainId],
        templateIds,
        scanAll,
        scheduledDate
      });
      return response.data;
    } catch (error) {
      console.error('Error scheduling scan:', error);
      throw new Error('Failed to schedule scan.');
    }
  },

  scanAll: async (domains: string[]): Promise<void> => {
    try {
      await axiosInstance.post('/v1/scans/all', { domains });
    } catch (error) {
      console.error('Error scanning all domains:', error);
      throw new Error('Failed to initiate scan for all domains.');
    }
  },

  getAllScans: async (): Promise<Scan[]> => {
    try {
      const response = await axiosInstance.get('/v1/scans');
      return response.data;
    } catch (error) {
      console.error('Error fetching all scans:', error);
      throw new Error('Failed to fetch all scans.');
    }
  },

  getMultiScans: async (): Promise<Scan[]> => {
    try {
      const response = await axiosInstance.get('/v1/scans/multi');
      return response.data;
    } catch (error) {
      console.error('Error fetching multi scans:', error);
      throw new Error('Failed to fetch multi scans.');
    }
  },

  scanDomains: async (domainIds: string[], templateIds: string[], scanAllNuclei: boolean = false): Promise<void> => {
    try {
      await axiosInstance.post('/v1/scans', {
        domainIds,
        templateIds,
        scanAllNuclei
      });
    } catch (error) {
      console.error('Error scanning domains:', error);
      throw new Error('Failed to scan domains.');
    }
  },

  deleteScheduledScan: async (id: string): Promise<void> => {
    try {
      await axiosInstance.delete(`/v1/scans/schedule/${id}`);
    } catch (error) {
      console.error('Error deleting scheduled scan:', error);
      throw new Error('Failed to delete scheduled scan.');
    }
  },

  getScanById: async (id: string[]): Promise<any> => {
    try {
      const response = await axiosInstance.get(`/v1/scans/${id}`);
      return response.data;
    } catch (error) {
      console.error(`Error fetching scan by ID (${id}):`, error);
      throw new Error(`Failed to fetch scan with ID: ${id}`);
    }
  }
};
