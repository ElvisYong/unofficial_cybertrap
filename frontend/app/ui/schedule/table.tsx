"use client";
import { useEffect, useState } from 'react';
import { XMarkIcon } from '@heroicons/react/24/outline';
import { Domain, ScheduledScanResponse, Template } from '@/app/types';
import { scanApi } from '@/api/scans';
import { domainApi } from '@/api/domains';
import { templateApi } from '@/api/templates';

export default function ScheduleScanTable() {
  const [scans, setScans] = useState<ScheduledScanResponse[]>([]);
  const [domains, setDomains] = useState<Domain[]>([]);
  const [templates, setTemplates] = useState<Template[]>([]);

  const fetchDomains = async () => {
    try {
      const data = await domainApi.getAllDomains();
      setDomains(data);
    } catch (error) {
      console.error('Error fetching domains:', error);
    }
  };

  const fetchTemplates = async () => {
    try {
      const data = await templateApi.getAllTemplates();
      setTemplates(data);
    } catch (error) {
      console.error('Error fetching templates:', error);
    }
  };

  const fetchScheduledScans = async () => {
    try {
      const data = await scanApi.getScheduledScans();
      setScans(data);
    } catch (error) {
      console.error('Error fetching scheduled scans:', error);
    }
  };

  useEffect(() => {
    fetchScheduledScans();
    fetchDomains();
    fetchTemplates();
  }, []);

  const getDomainNameById = (domainID: string, scanAll?: boolean) => {
    if (scanAll) return 'All Domains';
    const domain = domains.find(d => d.id === domainID);
    return domain ? domain.domain : 'Unknown Domain';
  };

  const getTemplateNamesByIds = (templateIDs: string[], scanAll?: boolean) => {
    if (scanAll) return 'All Templates';
    if (!templateIDs || templateIDs.length === 0) {
      return 'No templates selected';
    }
    const matchedTemplates = templates.filter(t => templateIDs.includes(t.id));
    return matchedTemplates.map(t => t.name).join(', ');
  };

  const handleDelete = async (scanID: string) => {
    try {
      await scanApi.deleteScheduledScan(scanID);
      setScans(prevScans => prevScans.filter(scan => scan.id !== scanID));
      console.log('Scan deleted successfully');
    } catch (error) {
      console.error('Error deleting scheduled scan:', error);
    }
  };

  return (
    <div className="overflow-x-auto">
      <table className="min-w-full border-collapse table-auto">
        <thead>
          <tr className="bg-gray-100 text-left">
            <th className="px-4 py-2">Domain</th>
            <th className="px-4 py-2">Templates</th>
            <th className="px-4 py-2">Scheduled Date</th>
            <th className="px-4 py-2">Actions</th>
          </tr>
        </thead>
        <tbody>
          {scans && scans.map((scan, index) => (
            <tr key={index} className="border-t">
              <td className="px-4 py-2">{getDomainNameById(scan.domainId, scan.scanAll)}</td>
              <td className="px-4 py-2">{getTemplateNamesByIds(scan.templatesIds, scan.scanAll)}</td>
              <td className="px-4 py-2">{scan.scheduledDate}</td>
              <td className="px-4 py-2 flex justify-center">
                <button
                  onClick={() => handleDelete(scan.id)}
                  className="bg-green-600 hover:bg-green-500 text-white px-3 py-1 rounded text-sm flex items-center gap-1"
                  title="Delete"
                >
                  <XMarkIcon className="h-4 w-4 text-white" />
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
