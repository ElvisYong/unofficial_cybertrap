"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
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
interface Domain {
    ID: string;
    Domain: string;
    UploadedAt: string;
    UserID: string; 
  }

export default function SelectScan() {
    const [templates, setTemplates] = useState<Template[]>([]);
    const [selectedTemplates, setSelectedTemplates] = useState<string[]>([]);
    const [scanAllTemplates, setScanAllTemplates] = useState(false);
    const [scanAllNuclei, setScanAllNuclei] = useState(true);
    const [target, setTarget] = useState("");
    const [scanName, setScanName] = useState("");
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
    }, [toast]); 

    const handleTemplateSelection = (templateId: string) => {
        setSelectedTemplates(prev => 
            prev.includes(templateId)
                ? prev.filter(id => id !== templateId)
                : [...prev, templateId]
        );
    };
    
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
        const domain = domains.find(d => d.ID === domainID);
        return domain ? domain.Domain : 'Unknown Domain';
    };

    const handleSubmit = async (event: React.FormEvent) => {
        event.preventDefault();
    
        const domainId = target;
    
        if (!domainId) {
            toast({
                title: "Error",
                description: "Invalid target domain.",
                variant: "destructive",
            });
            return;
        }
    
        const templateIds = scanAllTemplates ? [] : selectedTemplates;
        // const domainIdScanAll = templates.length > 0 ? templates[0].ID : "";

        const domainIdScanAll = target;

        console.log('template', templateIds);
        console.log('DIF', domainIdScanAll);
        // console.log('name', domainName)
    
        try {
            const requestBody = {
                domainId: domainIdScanAll,
                templateIds,
                scanAllNuclei,
                scanName,
            };
        
            console.log('Request Body:', JSON.stringify(requestBody, null, 2));
        
            const response = await fetch(`${BASE_URL}/v1/scans`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(requestBody),
            });
        
            console.log('Response Status:', response.status);
            
            let responseData;
        
            // Check if response is ok and has a body
            if (response.ok) {
                const contentType = response.headers.get('Content-Type');
                if (contentType && contentType.includes('application/json')) {
                    responseData = await response.json();
                } else {
                    responseData = {}; // Handle the case where there's no JSON
                }
        
                console.log('Response Data:', responseData);
                
                toast({
                    title: "Success",
                    description: "Scan initiated successfully.",
                });
        
                setTimeout(() => {
                    router.push("/dashboard/scans");
                }, 2000); // 2 second delay
            } else {
                // If response is not ok, attempt to read the error message
                const errorText = await response.text(); // Use text() to get the raw response
                const errorMessage = errorText ? errorText : 'Failed to initiate scan';
                throw new Error(errorMessage);
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
        <div className="flex items-center justify-center min-h-screen bg-gray-100">
            <div className="p-8 bg-white shadow-lg rounded-md flex-1 flex flex-col max-w-4xl">
                <h2 className="text-2xl font-bold mb-4">Select Scan Templates</h2>
                <p className="mb-4 text-gray-600">Target: {target}</p>
                <form onSubmit={handleSubmit} className="space-y-4 flex-grow">
                    <div className="space-y-4">
                        <div>
                            <label htmlFor="scanName" className="block text-sm font-medium text-gray-700 mb-1">
                                Scan Name
                            </label>
                            <Input
                                id="scanName"
                                type="text"
                                value={scanName}
                                onChange={(e) => setScanName(e.target.value)}
                                placeholder="Enter scan name"
                                className="w-full"
                                required
                            />
                        </div>
                        {/* {templates.map(template => (
                            <div key={template.ID} className="flex items-center space-x-3">
                                <Checkbox
                                    id={template.ID}
                                    checked={selectedTemplates.includes(template.ID)}
                                    onCheckedChange={() => handleTemplateSelection(template.ID)}
                                />
                                <label htmlFor={template.ID} className="text-gray-700">{template.Name}</label>
                            </div>
                        ))} */}
                        <div className="flex items-center space-x-3">
                            <Checkbox
                                id="Templates"
                                checked={scanAllTemplates}
                                onCheckedChange={(checked) => setScanAllTemplates(checked as boolean)}
                            />
                            <label htmlFor="scanAllTemplates" className="text-gray-700">Scan All Templates</label>
                        </div>
                    </div>
                    <Button
                        type="submit"
                        className="w-full py-2 mt-4 text-white bg-green-600 rounded-md hover:bg-green-700"
                        disabled={!scanName.trim() || (!scanAllTemplates && selectedTemplates.length === 0)}
                    >
                        Start Scan
                    </Button>
                </form>
            </div>
            <Toaster />
        </div>
    );
}