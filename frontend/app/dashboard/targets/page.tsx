"use client";

import { useState, useEffect } from 'react';
import TargetsTable from "@/app/ui/targets/table";
import { Dialog, DialogTrigger } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { PlusCircleIcon } from '@heroicons/react/24/outline';
import TargetModal from '../../ui/components/target-modal';
import { Input } from '@/components/ui/input';
import { BASE_URL } from '@/data';
import { Domain } from '@/app/types';
import { useToast } from "@/components/ui/use-toast";
import { Toaster } from "@/components/ui/toaster";

export default function Targets() : JSX.Element {
  const [isModalOpen, setIsModalOpen] = useState<boolean>(false);
  const [domains, setDomains] = useState<Domain[]>([]);
  const [searchTerm, setSearchTerm] = useState<string>('');
  const [isScanning, setIsScanning] = useState<boolean>(false);
  const { toast } = useToast();

  const handleOpenModal = () => setIsModalOpen(true);
  const handleCloseModal = () => setIsModalOpen(false);

  const fetchDomains = async () => {
    const endpoint = `${BASE_URL}/v1/domains`;
    try {
      const response = await fetch(endpoint);
      if (!response.ok) {
        throw new Error('Network response was not ok');
      }
      const data: Domain[] = await response.json();
      const sortedDomains = data.sort((a, b) => 
        new Date(b.uploadedAt).getTime() - new Date(a.uploadedAt).getTime()
      );
      setDomains(sortedDomains);
    } catch (error) {
      console.error('Error fetching domains:', error);
    }
  };

  useEffect(() => {
    fetchDomains();
  }, []);

  const handleTargetAdded = () => {
    fetchDomains();
    handleCloseModal();
  };

  // Filter the domains based on the search term
  const filteredDomains = domains.filter(domain =>
    domain.domain.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const handleScanAll = async () => {
    setIsScanning(true);
    const domainNames = domains.map(domain => domain.domain);
    
    try {
      // testing with mock api, to update the url
      const response = await fetch(`${BASE_URL}/v1/scan/all`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ domains: domainNames }),
      });

      if (response.ok) {
        toast({
          title: "Success",
          description: "Scan initiated for all targets.",
        });
      } else {
        throw new Error('Failed to initiate scan for all targets');
      }
    } catch (error) {
      console.error('Error initiating scan for all targets:', error);
      toast({
        title: "Error",
        description: "Failed to initiate scan for all targets. Please try again.",
        variant: "destructive",
      });
    } finally {
      setIsScanning(false);
    }
  };

  return (
    <div className="container mx-auto px-4">
      <h1 className="text-2xl font-bold mb-4">Targets</h1>
      
      <div className="space-y-4">
        <div className="flex items-center space-x-4">
          <Input
            placeholder="Search domain names..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="flex-grow"
          />
          <Dialog open={isModalOpen} onOpenChange={setIsModalOpen}>
            <DialogTrigger asChild>
              <Button
                onClick={handleOpenModal}
                className="bg-green-600 text-white px-4 py-2 rounded flex items-center gap-2 whitespace-nowrap"
              >
                <PlusCircleIcon className="h-4 w-4 text-white" />
                <span>Add Target</span>
              </Button>
            </DialogTrigger>
            <TargetModal isOpen={isModalOpen} onClose={handleCloseModal} onTargetAdded={handleTargetAdded} />
          </Dialog>
        </div>

        <div className="flex items-center space-x-4">
          <span className="text-sm font-medium whitespace-nowrap">Choose to Scan All Domains:</span>
          <Button
            onClick={handleScanAll}
            disabled={isScanning || domains.length === 0}
            className="bg-green-600 text-white px-4 py-2 rounded whitespace-nowrap"
          >
            {isScanning ? 'Scanning...' : 'Scan All Targets'}
          </Button>
        </div>
      </div>

      <div className="mt-4">
        <TargetsTable domains={filteredDomains} />
      </div>

      <Toaster/>

    </div>
  );
}