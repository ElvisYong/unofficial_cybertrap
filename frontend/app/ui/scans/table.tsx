'use client'

import { formatDateToLocal } from '@/app/lib/utils'
import { useRouter } from 'next/navigation'
import { InformationCircleIcon } from '@heroicons/react/24/outline'
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
import { Domain, Scan } from '@/app/types'
import { format } from 'date-fns'; // Import date-fns for formatting
import { domainApi } from '@/api/domains';
import { scanApi } from '@/api/scans'; // Import scanApi

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
    key: 'scanDate',
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
      const data = await scanApi.getAllScans(); // Use scanApi to fetch scans
      const sortedScans = data.sort((a: Scan, b: Scan) => 
        new Date(b.scanDate).getTime() - new Date(a.scanDate).getTime()
      )
  
      setScans(sortedScans)
      setFilteredScans(sortedScans)
    } catch (error) {
      console.error('Error fetching scans:', error)
    }
  }  

  const fetchDomains = async () => {
    try {
      const data = await domainApi.getAllDomains();
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
      if (key === 'scanDate') {
        const aDate = new Date(a.scanDate).getTime()
        const bDate = new Date(b.scanDate).getTime()
        return direction === 'asc' ? aDate - bDate : bDate - aDate
      }
      
      const aValue = String(a[key as keyof Scan])
      const bValue = String(b[key as keyof Scan])
      return direction === 'asc' 
        ? aValue.localeCompare(bValue)
        : bValue.localeCompare(aValue)
    })
    setFilteredScans(sortedScans)
  }
  
  const applyFilters = (sortedScans: Scan[]) => {
    let filtered = sortedScans;

    if (filters.domain && filters.domain.trim() !== '') {
      const lowercaseDomain = filters.domain.toLowerCase().trim();
      filtered = filtered.filter(scan => (scan.domain || '').toLowerCase().includes(lowercaseDomain));
    }
  
    if (filters.templateID && filters.templateID.trim() !== '') {
      const lowercaseTemplateID = filters.templateID.toLowerCase().trim();
      filtered = filtered.filter(scan => scan.templateIds.some(templateID => 
        templateID.toLowerCase().includes(lowercaseTemplateID)
      ));
    }
  
    if (filters.status && filters.status.trim() !== '') {
      const lowercaseStatus = filters.status.toLowerCase().trim();
      filtered = filtered.filter(scan => scan.status.toLowerCase().includes(lowercaseStatus));
    }
  
    setFilteredScans(filtered);
    setCurrentPage(1);
  };

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
    const domain = domains.find(d => d.id === domainID);
    return domain ? domain.domain : 'Unknown Domain';
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return format(date, 'Pp'); // Use date-fns for consistent formatting
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
                <th scope="col" className="px-4 py-5 font-medium sm:pl-6">Domain</th>
                <th scope="col" className="px-3 py-5 font-medium">Template IDs</th>
                <th scope="col" className="px-3 py-5 font-medium">
                  <SortButton
                    sortKey="scanDate"
                    sortConfig={sortConfig}
                    onSort={handleSort}
                    label="Scan Date"
                  />
                </th>
                <th scope="col" className="px-3 py-5 font-medium">Status</th>
                <th scope="col" className="relative py-3 pl-6 pr-3">Action</th>
              </tr>
            </thead>
            <tbody className="bg-white">
              {paginatedScans.map((scan) => (
                <tr key={scan.id} className="w-full border-b py-3 text-sm last-of-type:border-none">
                  <td className="whitespace-nowrap py-3 pl-6 pr-3">{scan.domain || getDomainNameById(scan.domainId)}</td>
                  <td className="whitespace-nowrap px-3 py-3">{scan.templateIds.join(', ') || 'All Github Default Template'}</td>
                  <td className="whitespace-nowrap px-3 py-3">{scan.status.toLowerCase() === 'completed' ? formatDate(scan.scanDate) : 'N/A'}</td>
                  <td className="whitespace-nowrap px-3 py-3">{getStatusBadge(scan.status)}</td>
                  <td className="whitespace-nowrap py-3 pl-6 pr-3">
                    <div className="flex space-x-4">
                      <button
                        onClick={() => handleViewDetails(scan.id)}
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
