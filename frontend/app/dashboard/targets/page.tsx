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

export default function Targets() : JSX.Element {
  const [isModalOpen, setIsModalOpen] = useState<boolean>(false);
  const [domains, setDomains] = useState<Domain[]>([]);
  const [searchTerm, setSearchTerm] = useState<string>('');

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

  return (
    <div>
      <b>Targets</b>
      
      <div className="flex justify-end my-4">
        <Input
            placeholder="Search domain names..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full px-4 py-2 mb-4 border rounded"
          />

        <Dialog open={isModalOpen} onOpenChange={setIsModalOpen}>
          <DialogTrigger asChild>
            <Button
              onClick={handleOpenModal}
              className="bg-green-600 text-white px-4 py-2 rounded flex items-center gap-2"
            >
              <PlusCircleIcon className="h-4 w-4 text-white" />
              <span>Add Target</span>
            </Button>
          </DialogTrigger>
          <TargetModal isOpen={isModalOpen} onClose={handleCloseModal} onTargetAdded={handleTargetAdded} />
        </Dialog>
      </div>

      <TargetsTable domains={filteredDomains} />
    </div>
  );
}
