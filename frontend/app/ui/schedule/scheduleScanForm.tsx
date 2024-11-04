'use client';

import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import { Popover, PopoverTrigger, PopoverContent } from "@/components/ui/popover";
import { format } from 'date-fns';
import { BASE_URL } from '@/data';
import { useState, useEffect } from "react";
import { Domain, Template } from "@/app/types";
import TemplateSearch from "./templateSearch";
import DomainSearch from "./domainSearch";
import { scanApi } from '@/api/scans';
import { domainApi } from '@/api/domains';
import { templateApi } from '@/api/templates';


type ScheduleScanFormProps = {
  onSubmit: (formData: any) => void;
};

export default function ScheduleScanForm({ onSubmit }: ScheduleScanFormProps) {
  const [selectedDomain, setSelectedDomain] = useState<Domain | null>(null);
  const [selectedTemplates, setSelectedTemplates] = useState<Template[]>([]);
  const [scanDate, setScanDate] = useState<Date | null>(null);

  //domains
  const [domains, setDomains] = useState<Domain[]>([]);
  const fetchDomains = async () => {
    try {
      const data = await domainApi.getAllDomains();
      setDomains(data);
    } catch (error) {
      console.error('Error fetching domains:', error);
    }
  };
  useEffect(() => {
    fetchDomains();
  }, []);

  //templates
  const [templates, setTemplates] = useState<Template[]>([]);
  const fetchTemplates = async () => {
    try {
      const data = await templateApi.getAllTemplates();
      setTemplates(data);
    } catch (error) {
      console.error('Error fetching templates:', error);
    }
  }
  useEffect(() => {
    fetchTemplates();
  }, []);

  const handleTemplateSelect = (template: any) => {
    setSelectedTemplates((prev) => [...prev, template]);
  };

  const handleTemplateDeselect = (template: any) => {
    setSelectedTemplates((prev) =>
      prev.filter((t) => t.id !== template.id)
    );
  };

  // submit form
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!selectedDomain?.id || !scanDate) {
      console.error('Missing required fields');
      return;
    }

    try {
      await scanApi.scheduleScan(selectedDomain.id, selectedTemplates.map(template => template.id), format(scanDate, 'yyyy-MM-dd'));

      console.log('Scan scheduled successfully');
      setSelectedDomain(null);
      setSelectedTemplates([]);
      setScanDate(null);
    } catch (error) {
      console.error('Error submitting form:', error);
    }
  };


  return (
    <form onSubmit={handleSubmit} className="space-y-6 mx-lg">
      {/* Domain Input */}
      <div className="space-y-2">
        <label htmlFor="domain" className="block text-sm font-medium">
          Domain
        </label>
        <DomainSearch
          domains={domains}
          selectedDomain={selectedDomain}
          onDomainSelect={setSelectedDomain}
        />
      </div>

      {/* Multi-Select for Template IDs */}
      <div className="space-y-2">
        <label htmlFor="templates" className="block text-sm font-medium">
          Select Templates
        </label>
        <TemplateSearch
          templates={templates}
          selectedTemplates={selectedTemplates}
          onTemplateSelect={handleTemplateSelect}
          onTemplateDeselect={handleTemplateDeselect}
        />
        <p className="mt-2 text-sm">Selected Templates: {selectedTemplates.map(t => t.name).join(', ') || 'None'}</p>

      </div>

      {/* Scan Date Picker */}
      <div className="space-y-2">
        <label htmlFor="scanDate" className="block text-sm font-medium">
          Select Scan Date
        </label>
        <Popover>
          <PopoverTrigger className="border px-4 py-2 rounded cursor-pointer w-full text-left">
            {scanDate ? format(scanDate, 'PPP') : 'Pick a date'}
          </PopoverTrigger>
          <PopoverContent className="p-0 w-auto">
            <Calendar
              mode="single"
              selected={scanDate as Date}
              onSelect={(date: Date | undefined) => setScanDate(date ?? null)}
              initialFocus
            />
          </PopoverContent>
        </Popover>
      </div>

      {/* Submit Button */}
      <Button type="submit" className="w-full bg-green-600 hover:bg-green-500 text-white">
        Schedule Scan
      </Button>
    </form>
  );
}