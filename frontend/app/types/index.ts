export interface Domain {
  id: string;
  domain: string;
  uploadedAt: string;
  userId: string; 
}

export interface Template {
  id: string;
  templateId: string;
  name: string;
  description: string;
  s3Url: string;
  metadata: null | any;
  type: string;
  CreatedAt: string;
}

export interface Scan {
  id: string;
  domainId: string;
  domain: string;
  templateIds: string[];
  scanDate: string;
  status: string;
  error: string | null;
  s3ResultURL: string | null;
}

export interface ScheduledScanResponse {
  id: string;
  domainId: string;
  templatesIds: string[];
  scheduledDate: string;
}