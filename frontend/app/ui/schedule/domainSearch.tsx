import { useState } from 'react';
import { Popover, PopoverTrigger, PopoverContent } from "@/components/ui/popover";
import { Input } from '@/components/ui/input';
import { Domain } from '@/app/types';

type DomainSearchProps = {
  domains: Domain[];
  selectedDomain: Domain | null;
  onDomainSelect: (domain: Domain) => void;
};

export default function DomainSearch({ domains, selectedDomain, onDomainSelect }: DomainSearchProps) {
  const [searchTerm, setSearchTerm] = useState<string>('');

  // Filter domains based on the search term
  const filteredDomains = domains.filter(domain =>
    domain.domain.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <Popover>
      <PopoverTrigger className="border px-4 py-2 rounded cursor-pointer w-full text-left">
        {selectedDomain?.domain || 'Select Domain'}
      </PopoverTrigger>
      <PopoverContent className="w-72 p-4">
        <Input
          placeholder="Search Domains"
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="mb-4"
        />
        <div className="max-h-48 overflow-y-auto">
          {filteredDomains.length > 0 ? (
            filteredDomains.map((domain) => (
              <div
                key={domain.id}
                onClick={() => onDomainSelect(domain)}
                className={`p-2 hover:bg-gray-100 cursor-pointer ${selectedDomain?.id === domain.id ? 'bg-gray-200' : ''}`}
              >
                {domain.domain}
              </div>
            ))
          ) : (
            <p className="text-sm text-gray-500">No matching domains found</p>
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
}