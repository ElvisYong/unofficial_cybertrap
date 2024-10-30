'use client'

import { useState, useRef } from 'react';
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Toaster } from "@/components/ui/toaster";
import { useToast } from "@/hooks/use-toast";
import { BASE_URL } from '@/data';

export default function Component({ isOpen = false, onClose = () => {}, onTargetAdded = () => {} }) {
  const [targetName, setTargetName] = useState('');
  const [file, setFile] = useState<File | null>(null);
  const { toast } = useToast();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files) {
      setFile(e.target.files[0]);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!targetName.trim() && !file) {
      toast({
        title: "Error",
        description: "Please enter a target name or upload a file.",
        variant: "destructive",
      });
      return;
    }

    try {
      if (targetName.trim()) {
        await createDomain(targetName);
      }

      if (file) {
        await uploadFile(file);
      }

      onTargetAdded(); // Notify parent component of the new target(s)
      onClose();
      toast({
        title: "Success",
        description: "Target(s) added successfully!",
      });
    } catch (error) {
      console.error('Error:', error);
      toast({
        title: "Error",
        description: "An unexpected error occurred! Please try again.",
        variant: "destructive",
      });
    }
  };

  const createDomain = async (domain: string) => {
    const response = await fetch(`${BASE_URL}/v1/domains/create?domain=${encodeURIComponent(domain)}`, {
      method: 'POST',
    });

    if (!response.ok) {
      throw new Error(`Failed to create domain: ${domain}`);
    }
  };

  const uploadFile = async (file: File) => {
    const formData = new FormData();
    formData.append('file', file);

    const response = await fetch(`${BASE_URL}/v1/domains/upload-txt`, {
      method: 'POST',
      body: formData,
    });

    if (!response.ok) {
      throw new Error('Failed to upload file');
    }
  };

  return (
    <>
      <Dialog open={isOpen} onOpenChange={onClose}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Add Target</DialogTitle>
            <DialogDescription>
              Enter a single target name or upload a .txt file containing multiple domain names.
            </DialogDescription>
          </DialogHeader>
          <form className="mt-2" onSubmit={handleSubmit}>
            <div className="mb-4">
              <Label htmlFor="targetName">Target Name</Label>
              <Input
                id="targetName"
                type="text"
                placeholder="grab.com"
                value={targetName}
                className='focus:ring-green-500 focus:border-green-500'
                onChange={(e) => setTargetName(e.target.value)}
              />
            </div>
            <div className="mb-4">
              <Label htmlFor="domainFile">Or upload a .txt file</Label>
              <Input
                id="domainFile"
                type="file"
                accept=".txt"
                ref={fileInputRef}
                onChange={handleFileChange}
              />
            </div>
            <div className="flex justify-end">
              <Button type="submit"
                className="bg-green-600 text-white px-4 py-2 rounded-md hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500"
              >
                Add Target(s)
              </Button>
            </div>
          </form>
          <DialogFooter className="sm:justify-start">
            <DialogClose asChild>
            </DialogClose>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <Toaster />
    </>
  );
}