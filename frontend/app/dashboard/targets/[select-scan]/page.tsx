"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { useToast } from "@/components/ui/use-toast";
import { Toaster } from "@/components/ui/toaster";
import { BASE_URL } from "@/data";

interface Template {
  ID: string;
  TemplateID: string;
  Name: string;
  Description: string;
  S3URL: string;
  Metadata: null | any;
  Type: string;
  CreatedAt: string;
}

export default function SelectScan() {
    const [templates, setTemplates] = useState<Template[]>([]);
    const [selectedTemplates, setSelectedTemplates] = useState<string[]>([]);
    const [target, setTarget] = useState("");
    const router = useRouter();
    const { toast } = useToast();

    useEffect(() => {
        const targetFromUrl = new URLSearchParams(window.location.search).get("target");
        if (targetFromUrl) {
            setTarget(targetFromUrl);
        }

        // Fetch templates
        fetch(`${BASE_URL}/v1/templates`)
            .then(response => response.json())
            .then(data => setTemplates(data))
            .catch(error => {
                console.error('Error fetching templates:', error);
                toast({
                    title: "Error",
                    description: "Failed to fetch templates. Please try again.",
                    variant: "destructive",
                });
            });
    }, []); 

    const handleTemplateSelection = (templateId: string) => {
        setSelectedTemplates(prev => 
            prev.includes(templateId)
                ? prev.filter(id => id !== templateId)
                : [...prev, templateId]
        );
    };

    const handleSubmit = async (event: React.FormEvent) => {
        event.preventDefault();
        if (selectedTemplates.length === 0) {
            toast({
                title: "Error",
                description: "Please select at least one template.",
                variant: "destructive",
            });
            return;
        }

        try {
            const response = await fetch(`${BASE_URL}/v1/scans`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    DomainID: target,
                    TemplateIDs: selectedTemplates
                }),
            });

            if (response.ok) {
                toast({
                    title: "Success",
                    description: "Scan initiated successfully.",
                });
                // Optionally, redirect to a results page or dashboard
                // router.push('/scan-results');
            } else {
                throw new Error('Failed to initiate scan');
            }
        } catch (error) {
            console.error('Error initiating scan:', error);
            toast({
                title: "Error",
                description: "Failed to initiate scan. Please try again.",
                variant: "destructive",
            });
        }
    };

    return (
        <>
            <div className="flex items-center justify-center min-h-screen bg-gray-100">
                <div className="p-8 bg-white shadow-lg rounded-md flex-1 flex flex-col max-w-4xl">
                    <h2 className="text-2xl font-bold mb-4">Select Scan Templates</h2>
                    <p className="mb-4 text-gray-600">Target: {target}</p>
                    <form onSubmit={handleSubmit} className="space-y-4 flex-grow">
                        <div className="space-y-4">
                            {templates.map(template => (
                                <div key={template.ID} className="flex items-center space-x-3">
                                    <Checkbox
                                        id={template.ID}
                                        checked={selectedTemplates.includes(template.ID)}
                                        onCheckedChange={() => handleTemplateSelection(template.ID)}
                                    />
                                    <label htmlFor={template.ID} className="text-gray-700">{template.Name}</label>
                                </div>
                            ))}
                        </div>
                        <Button
                            type="submit"
                            className="w-full py-2 mt-4 text-white bg-green-600 rounded-md hover:bg-green-700"
                            disabled={selectedTemplates.length === 0}
                        >
                            Start Scan
                        </Button>
                    </form>
                </div>
            </div>
            <Toaster />
        </>
    );
}