import { useState } from 'react';
import { Popover, PopoverTrigger, PopoverContent } from "@/components/ui/popover";
import { Input } from '@/components/ui/input';
import { Domain } from '@/app/types';
import { Button } from '@/components/ui/button';

type DomainSearchProps = {
  domains: Domain[];
  selectedDomain: Domain | null;
  onDomainSelect: (domain: Domain | null) => void;
  onScanAllChange: (scanAll: boolean) => void;
  isScanAllTemplates: boolean;
};

export default function DomainSearch({ 
  domains, 
  selectedDomain, 
  onDomainSelect, 
  onScanAllChange,
  isScanAllTemplates 
}: DomainSearchProps) {
  const [searchTerm, setSearchTerm] = useState<string>('');
  const [isOpen, setIsOpen] = useState(false);
  const [isScanAllDomains, setIsScanAllDomains] = useState(false);

  // Filter domains based on the search term
  const filteredDomains = domains ? domains.filter(domain =>
    domain.domain.toLowerCase().includes(searchTerm.toLowerCase())
  ): [];

  const handleScanAllToggle = () => {
    setIsScanAllDomains(!isScanAllDomains);
    onScanAllChange(!isScanAllDomains);
    onDomainSelect(null);  // Clear selected domain when scanning all
    setIsOpen(false);
  };

  return (
    <Popover open={isOpen} onOpenChange={setIsOpen}>
      <PopoverTrigger 
        className="border px-4 py-2 rounded cursor-pointer w-full text-left"
        disabled={isScanAllTemplates}  // Disable when all templates are selected
      >
        {isScanAllDomains 
          ? 'All Domains Selected'
          : selectedDomain?.domain || 'Select Domain'}
      </PopoverTrigger>
      <PopoverContent className="w-72 p-4">
        <Input
          placeholder="Search Domains"
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="mb-4"
        />
        <div className="flex justify-between mb-2">
          <Button 
            variant={isScanAllDomains ? "destructive" : "default"} 
            size="sm" 
            onClick={handleScanAllToggle}
            className="w-full"
          >
            {isScanAllDomains ? 'Deselect All' : 'Select All Domains'}
          </Button>
        </div>
        <div className="max-h-48 overflow-y-auto">
          {filteredDomains.length > 0 ? (
            filteredDomains.map((domain) => (
              <div
                key={domain.id}
                onClick={() => {
                  onDomainSelect(domain);
                  setIsOpen(false);
                }}
                className={`p-2 hover:bg-gray-100 cursor-pointer ${
                  selectedDomain?.id === domain.id ? 'bg-gray-200' : ''
                }`}
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