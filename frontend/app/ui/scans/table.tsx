'use client'

import { formatDateToLocal } from '@/app/lib/utils'
import { useRouter } from 'next/navigation'
import { InformationCircleIcon, BoltIcon } from '@heroicons/react/24/outline'
import { useState, useEffect } from 'react'
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination"
import FilterByString from '@/components/ui/filterString'
import FilterByDropdown from '@/components/ui/filterDropdown'
import SortButton from '@/components/ui/sortButton'
import { BASE_URL } from '@/data'

type Scan = {
  ID: string
  DomainID: string
  Domain: string
  TemplateIDs: string[]
  ScanDate: string
  Status: string
  Error: string | null
  S3ResultURL: string | null
}

interface Domain {
  ID: string;
  Domain: string;
  UploadedAt: string;
  UserID: string; 
}

export default function ScanResultsTable() {
  const [scans, setScans] = useState<Scan[]>([])
  const [filteredScans, setFilteredScans] = useState<Scan[]>([])
  const [currentPage, setCurrentPage] = useState(1)
  const [domains, setDomains] = useState<Domain[]>([])
  const itemsPerPage = 7
  const router = useRouter()

  const [filters, setFilters] = useState({
    domain: '',
    templateID: '',
    status: ''
  })
  const [sortConfig, setSortConfig] = useState({
    key: 'ScanDate',
    direction: 'desc'
  })

  useEffect(() => {
    fetchScans()
    fetchDomains()
  }, [])

  useEffect(() => {
    applyFilters(scans)
  }, [scans, filters])

  const fetchScans = async () => {
    try {
      const response = await fetch(`${BASE_URL}/v1/scans/`)
      if (!response.ok) {
        throw new Error('Failed to fetch scans')
      }
      const data = await response.json()
      console.log(data)

      const sortedScans = data.sort((a: Scan, b: Scan) => 
        new Date(b.ScanDate).getTime() - new Date(a.ScanDate).getTime()
      )
  
      setScans(sortedScans)
      setFilteredScans(sortedScans)
    } catch (error) {
      console.error('Error fetching scans:', error)
    }
  }  

  const fetchDomains = async () => {
    const endpoint = `${BASE_URL}/v1/domains`;
    try {
      const response = await fetch(endpoint);
      // console.log(response.json())
      if (!response.ok) {
        throw new Error('Network response was not ok');
      }
      const data: Domain[] = await response.json();
      setDomains(data);
    } catch (error) {
      console.error('Error fetching domains:', error);
    }
  };

  const handleSort = (key: string) => {
    let direction = 'asc'
    if (sortConfig.key === key && sortConfig.direction === 'asc') {
      direction = 'desc'
    }
    setSortConfig({ key, direction })

    const sortedScans = [...filteredScans].sort((a, b) => {
      const aValue = key === 'ScanDate' ? new Date(a[key]) : a[key]
      const bValue = key === 'ScanDate' ? new Date(b[key]) : b[key]
      return direction === 'asc' ? aValue - bValue : bValue - aValue
    })
    setFilteredScans(sortedScans)
  }
  
  const applyFilters = (sortedScans: Scan[]) => {
    let filtered = sortedScans
    if (filters.domain) {
      filtered = filtered.filter(scan =>
        scan.Domain.toLowerCase().includes(filters.domain.toLowerCase())
      )
    }

    if (filters.templateID) {
      filtered = filtered.filter(scan =>
        scan.TemplateIDs.some(templateID =>
          templateID.toLowerCase().includes(filters.templateID.toLowerCase())
        )
      )
    }

    if (filters.status) {
      filtered = filtered.filter(scan =>
        scan.Status.toLowerCase().includes(filters.status.toLowerCase())
      )
    }

    setFilteredScans(filtered)
    setCurrentPage(1) 
  }

  const handleFilter = (filterType: string, filterValue: string) => {
    setFilters(prevFilters => ({
      ...prevFilters,
      [filterType]: filterValue
    }))
  } 

  const handleViewDetails = (scanId: string) => {
    router.push(`/dashboard/scans/${encodeURIComponent(scanId)}`)
  }

  const resetFilters = () => {
    setFilters({
      domain: '',
      templateID: '',
      status: ''
    });
    setFilteredScans(scans);
    setCurrentPage(1);
  }

  const getStatusBadge = (status: string) => {
    switch (status.toLowerCase()) {
      case 'completed':
        return <span className="bg-green-500 text-white px-2 py-1 rounded">Completed</span>
      case 'in-progress':
        return <span className="bg-yellow-500 text-white px-2 py-1 rounded">In Progress</span>
      case 'pending':
        return <span className="bg-blue-500 text-white px-2 py-1 rounded">Pending</span>
      case 'failed':
        return <span className="bg-red-500 text-white px-2 py-1 rounded">Failed</span>
      default:
        return <span className="bg-gray-300 text-white px-2 py-1 rounded">Unknown</span>
    }
  }

  const pageCount = Math.ceil(filteredScans.length / itemsPerPage)
  const paginatedScans = filteredScans.slice(
    (currentPage - 1) * itemsPerPage,
    currentPage * itemsPerPage
  )

  const getDomainNameById = (domainID: string) => {
    const domain = domains.find(d => d.ID === domainID);
    return domain ? domain.Domain : 'Unknown Domain';
  };

  return (
    <div className="mt-6 flow-root">
      <div className="inline-block min-w-full align-middle">
        <div className="rounded-lg bg-gray-50 p-2 md:pt-0">
          <div>
            <FilterByString
              filterType="domain"
              placeholder="Filter by Domain"
              onFilter={handleFilter}
              value={filters.domain}
            />    
            <FilterByString
              filterType="templateID"
              placeholder="Filter by Template ID"
              onFilter={handleFilter}
              value={filters.templateID}
            />  
            <FilterByDropdown 
              filterType="status"
              placeholder="Filter By Status" 
              onFilter={handleFilter}
              value={filters.status}
            /> 
            <button
              onClick={resetFilters}
              className="bg-gray-600 text-white px-4 py-2 rounded"
            >
              Reset Filters
            </button>       
          </div>
          <table className="hidden min-w-full text-gray-900 md:table">
            <thead className="rounded-lg text-left text-sm font-normal">
              <tr>
                <th scope="col" className="px-4 py-5 font-medium sm:pl-6">
                Domain
                </th>
                <th scope="col" className="px-3 py-5 font-medium">
                  Template IDs
                </th>
                <th scope="col" className="px-3 py-5 font-medium">
                  <SortButton
                    sortKey="ScanDate"
                    sortConfig={sortConfig}
                    onSort={handleSort}
                    label="Scan Date"
                  />
                </th>
                <th scope="col" className="px-3 py-5 font-medium">
                  Status
                </th>
                <th scope="col" className="relative py-3 pl-6 pr-3">
                  Action
                </th>
              </tr>
            </thead>
            <tbody className="bg-white">
              {paginatedScans.map((scan) => (
                <tr
                  key={scan.ID}
                  className="w-full border-b py-3 text-sm last-of-type:border-none [&:first-child>td:first-child]:rounded-tl-lg [&:first-child>td:last-child]:rounded-tr-lg [&:last-child>td:first-child]:rounded-bl-lg [&:last-child>td:last-child]:rounded-br-lg"
                >
                  <td className="whitespace-nowrap py-3 pl-6 pr-3">
                    {scan.Domain || getDomainNameById(scan.DomainID)}
                  </td>
                  <td className="whitespace-nowrap px-3 py-3">
                    {scan.TemplateIDs.join(', ') || 'All Github Default Template'}
                  </td>
                  <td className="whitespace-nowrap px-3 py-3">
                    {scan.Status.toLowerCase() === 'completed' 
                      ? formatDateToLocal(scan.ScanDate)
                      : 'N/A'}
                  </td>
                  <td className="whitespace-nowrap px-3 py-3">
                    {getStatusBadge(scan.Status)}
                  </td>
                  <td className="whitespace-nowrap py-3 pl-6 pr-3">
                    <div className="flex space-x-4">
                      <button
                        onClick={() => handleViewDetails(scan.ID)}
                        className="bg-green-600 text-white px-4 py-2 rounded flex items-center gap-2"
                      >
                        <InformationCircleIcon className="h-4 w-4 text-white" />
                        <span>Details</span>
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
  )
}