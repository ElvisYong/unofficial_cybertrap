"use client";

import { formatDateToLocal } from '@/app/lib/utils';
import { useRouter } from 'next/navigation';
import { InformationCircleIcon, BoltIcon } from '@heroicons/react/24/outline';
import { useState } from 'react';
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";

interface Domain {
  ID: string;
  Domain: string;
  UploadedAt: string;
  UserID: string;
}

interface TargetsTableProps {
  domains: Domain[];
}

export default function TargetsTable({ domains }: TargetsTableProps) {
  const [currentPage, setCurrentPage] = useState(1);
  const router = useRouter();
  const itemsPerPage = 7;

  const handleViewDetails = (target: string) => {
    router.push(`/dashboard/scans/${encodeURIComponent(target)}`);
  };

  const selectScanEngine = (target: string) => {
    router.push(`/dashboard/targets/select-scan?target=${encodeURIComponent(target)}`);
  };

  const pageCount = Math.ceil(domains.length / itemsPerPage);
  const paginatedDomains = domains.slice(
    (currentPage - 1) * itemsPerPage,
    currentPage * itemsPerPage
  );

  return (
    <div className="mt-6 flow-root">
      <div className="inline-block min-w-full align-middle">
        <div className="rounded-lg bg-gray-50 p-2 md:pt-0">
          <div className="md:hidden">
            {paginatedDomains.map((domain) => (
              <div
                key={domain.ID}
                className="mb-2 w-full rounded-md bg-white p-4"
              >
              </div>
            ))}
          </div>

          <table className="hidden min-w-full text-gray-900 md:table">
            <thead className="rounded-lg text-left text-sm font-normal">
              <tr>
                <th scope="col" className="px-4 py-5 font-medium sm:pl-6">
                  Domain
                </th>
                <th scope="col" className="px-3 py-5 font-medium">
                  User ID
                </th>
                <th scope="col" className="px-3 py-5 font-medium">
                  Uploaded At
                </th>
                <th scope="col" className="relative py-3 pl-6 pr-3">
                  Action
                </th>
              </tr>
            </thead>
            <tbody className="bg-white">
              {paginatedDomains.map((domain) => (
                <tr
                  key={domain.ID}
                  className="w-full border-b py-3 text-sm last-of-type:border-none [&:first-child>td:first-child]:rounded-tl-lg [&:first-child>td:last-child]:rounded-tr-lg [&:last-child>td:first-child]:rounded-bl-lg [&:last-child>td:last-child]:rounded-br-lg"
                >
                  <td className="whitespace-nowrap py-3 pl-6 pr-3">
                    {domain.Domain}
                  </td>
                  <td className="whitespace-nowrap px-3 py-3">
                    {domain.UserID}
                  </td>
                  <td className="whitespace-nowrap px-3 py-3">
                    {formatDateToLocal(domain.UploadedAt)}
                  </td>
                  <td className="whitespace-nowrap py-3 pl-6 pr-3">
                    <div className="flex space-x-4">
                      <button
                        onClick={() => handleViewDetails(domain.ID)}
                        className="bg-green-600 hover:bg-green-500 text-white px-4 py-2 rounded flex items-center gap-2"
                      >
                        <InformationCircleIcon className="h-4 w-4 text-white" />
                        <span>Target Summary</span>
                      </button>
                      <button 
                        onClick={() => selectScanEngine(domain.ID)}
                        className="bg-green-600  hover:bg-green-500 text-white px-4 py-2 rounded flex items-center gap-2"
                      >
                        <BoltIcon className="h-4 w-4 text-white" />
                        <span>Initiate Scan</span>
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>

          <div className="mt-6">
            <Pagination>
              <PaginationContent>
                <PaginationItem>
                  <PaginationPrevious 
                    onClick={() => setCurrentPage(prev => Math.max(prev - 1, 1))}
                    className={currentPage === 1 ? 'pointer-events-none opacity-50' : ''}
                  />
                </PaginationItem>
                {[...Array(pageCount)].map((_, i) => (
                  <PaginationItem key={i}>
                    <PaginationLink
                      onClick={() => setCurrentPage(i + 1)}
                      isActive={currentPage === i + 1}
                    >
                      {i + 1}
                    </PaginationLink>
                  </PaginationItem>
                ))}
                <PaginationItem>
                  <PaginationNext 
                    onClick={() => setCurrentPage(prev => Math.min(prev + 1, pageCount))}
                    className={currentPage === pageCount ? 'pointer-events-none opacity-50' : ''}
                  />
                </PaginationItem>
              </PaginationContent>
            </Pagination>
          </div>
        </div>
      </div>
    </div>
  );
}
