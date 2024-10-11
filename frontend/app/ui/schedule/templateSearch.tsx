import { useState, useEffect } from 'react';
import { Popover, PopoverTrigger, PopoverContent } from "@/components/ui/popover";
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Template } from '@/app/types';

type TemplateSearchProps = {
  templates: Template[];
  selectedTemplates: Template[];
  onTemplateSelect: (template: Template) => void;
  onTemplateDeselect: (template: Template) => void;
};

export default function TemplateSearch({ templates, selectedTemplates, onTemplateSelect, onTemplateDeselect }: TemplateSearchProps) {
  const [searchTerm, setSearchTerm] = useState<string>('');
  const [allSelected, setAllSelected] = useState<boolean>(false);

  // Filter templates based on the search term
  const filteredTemplates = templates.filter(template =>
    template.name.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const isSelected = (template: Template) => selectedTemplates.some(t => t.id === template.id);
  
  // Check if all filtered templates are selected
  useEffect(() => {
    const allSelected = filteredTemplates.length > 0 && filteredTemplates.every(template => isSelected(template));
    setAllSelected(allSelected);
    console.log('selected', allSelected)
  }, [filteredTemplates, selectedTemplates]);

  // Toggle select/deselect all
  const handleToggleSelectAll = () => {
    if (allSelected) {
      // Deselect all filtered templates
      filteredTemplates.forEach(template => {
        if (isSelected(template)) {
          onTemplateDeselect(template);
        }
      });
    } else {
      // Select all filtered templates
      filteredTemplates.forEach(template => {
        if (!isSelected(template)) {
          onTemplateSelect(template);
        }
      });
    }
    setAllSelected(!allSelected);
  };

  return (
    <Popover>
      <PopoverTrigger className="border px-4 py-2 rounded cursor-pointer w-full text-left">
        {selectedTemplates.length > 0 ? `Selected (${selectedTemplates.length})` : 'Select Templates'}
      </PopoverTrigger>
      <PopoverContent className="w-72 p-4">
        <Input
          placeholder="Search Templates"
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="mb-4"
        />
        <div className="flex justify-between mb-2">
          <Button variant="default" size="sm" onClick={handleToggleSelectAll}>
            {allSelected ? 'Deselect All' : 'Select All'}
          </Button>
        </div>
        <div className="max-h-48 overflow-y-auto">

          {filteredTemplates.length > 0 ? (
            filteredTemplates.map((template) => (
              <div
                key={template.id}
                className="flex justify-between items-center p-2 hover:bg-gray-100 cursor-pointer"
              >
                <label className="flex items-center space-x-2">
                  <input
                    type="checkbox"
                    checked={isSelected(template)}
                    onChange={(e) =>
                      e.target.checked
                        ? onTemplateSelect(template)
                        : onTemplateDeselect(template)
                    }
                  />
                  <span>{template.name}</span>
                </label>
              </div>
            ))
          ) : (
            <p className="text-sm text-gray-500">No matching templates found</p>
          )}
        </div>

      </PopoverContent>
    </Popover>
  );
  }