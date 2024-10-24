"use client";
import { useEffect, useState } from 'react';
import { XMarkIcon } from '@heroicons/react/24/outline'; // Import the Heroicon
import { BASE_URL } from '@/data';
import { Domain, ScheduledScanResponse, Template } from '@/app/types';

export default function ScheduleScanTable() {
  const [scans, setScans] = useState<ScheduledScanResponse[]>([]);

  //domains
  const [domains, setDomains] = useState<Domain[]>([]);
  const fetchDomains = async () => {
    const endpoint = `${BASE_URL}/v1/domains`;
    try {
      const response = await fetch(endpoint);
      if (!response.ok) {
        throw new Error('Network response was not ok');
      }
      const data: Domain[] = await response.json();
      setDomains(data);
      console.log('domain', data);
    } catch (error) {
      console.error('Error fetching domains:', error);
    }
  };
  useEffect(() => {
    fetchDomains();
  }, []);

  // Function to get the domain name by ID
  const getDomainNameById = (domainID: string) => {
    const domain = domains.find(d => d.id === domainID);
    return domain ? domain.domain : 'Unknown Domain';
  };

  //templates
  const [templates, setTemplates] = useState<Template[]>([]);
  const fetchTemplates = async () => {
    const endpoint = `${BASE_URL}/v1/templates`;
    try {
      const response = await fetch(endpoint);
      if (!response.ok) {
        throw new Error('Network response was not ok');
      }
      const data: Template[] = await response.json();
      setTemplates(data);
      console.log('template', data);
    } catch (error) {
      console.error('Error fetching templates:', error);
    }
  }
  useEffect(() => {
    fetchTemplates();
  }, []);

  // Function to get the template names by their IDs
  const getTemplateNamesByIds = (templateIDs: string[]) => {
    if (!templateIDs || templateIDs.length === 0) {
      return 'null'; // Return 'null' if template IDs are not provided or empty
    }
    const matchedTemplates = templates.filter(t => templateIDs.includes(t.id));
    return matchedTemplates.map(t => t.name).join(', ');
  };

  // Fetch the scheduled scans when the component mounts
  const fetchScheduledScans = async () => {
    const endpoint = `${BASE_URL}/v1/scans/schedule`;
    try {
      const response = await fetch(endpoint);
      if (!response.ok) {
        throw new Error('Network response was not ok');
      }

      const data: ScheduledScanResponse[] = await response.json();
      setScans(data);
      console.log('scan', data);
    } catch (error) {
      console.error('Error fetching domains:', error);
    }
  };

  useEffect(() => {
    fetchScheduledScans();
    fetchDomains();
  }, []);


  // Function to delete a scheduled scan by its ID
  const handleDelete = async (scanID: string) => {
    try {
      console.log('Deleting scan:', scanID);
      const response = await fetch(`${BASE_URL}/v1/scans/schedule?ID=${scanID}`, {
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        throw new Error('Failed to delete scan');
      }

      // Update state to remove the deleted scan from the UI
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
              <td className="px-4 py-2">{getDomainNameById(scan.domainId)}</td>
              <td className="px-4 py-2">{getTemplateNamesByIds(scan.templatesIds)}</td>
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
