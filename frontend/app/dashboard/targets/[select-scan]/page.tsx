"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { BASE_URL } from "@/data";
import { domainApi } from "@/api/domains";
import { Domain, Template } from "@/app/types";
import { scanApi } from "@/api/scans";
import toast from "react-hot-toast";
import { templateApi } from "@/api/templates";

export default function SelectScan() {
    const [templates, setTemplates] = useState<Template[]>([]);
    const [selectedTemplates, setSelectedTemplates] = useState<string[]>([]);
    const [scanAllTemplates, setScanAllTemplates] = useState(false);
    const [scanAllNuclei, setScanAllNuclei] = useState(true);
    const [target, setTarget] = useState("");
    const [scanName, setScanName] = useState("");
    const router = useRouter();

    useEffect(() => {
        const targetFromUrl = new URLSearchParams(window.location.search).get("target");
        if (targetFromUrl) {
            setTarget(targetFromUrl);
        }

        // Fetch templates using templateApi
        const fetchTemplates = async () => {
            try {
                const data = await templateApi.getAllTemplates();
                setTemplates(data);
            } catch (error) {
                console.error('Error fetching templates:', error);
                toast.error("Failed to fetch templates. Please try again.");
            }
        };

        fetchTemplates();
    }, []);

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
        try {
            const data = await domainApi.getAllDomains();
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

    const handleSubmit = async (event: React.FormEvent) => {
        event.preventDefault();

        const domainId = target;

        if (!domainId) {
            toast.error("Invalid target domain.");
            return;
        }

        const templateIds = scanAllTemplates ? [] : selectedTemplates;
        const domainIdScanAll = target;

        console.log('template', templateIds);
        console.log('DIF', domainIdScanAll);

        try {
            const requestBody = {
                domainIds: [domainIdScanAll],
                templateIds,
                scanAllNuclei,
            };

            console.log('Request Body:', JSON.stringify(requestBody, null, 2));

            await scanApi.scanDomains(requestBody.domainIds, requestBody.templateIds, requestBody.scanAllNuclei);

            toast.success("Scan initiated successfully.");

            setTimeout(() => {
                router.push("/dashboard/scans");
            }, 2000); // 2 second delay
        } catch (error) {
            console.error('Error initiating scan:', error);
            toast.error("Failed to initiate scan. Please try again.");
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
                                onCheckedChange={(checked) => setScanAllTemplates(checked.valueOf() as boolean)}
                            />
                            <label htmlFor="Templates" className="text-gray-700">Scan All Templates</label>
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
        </div>
    );
}