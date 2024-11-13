'use client'
import React, { useState, useEffect } from 'react'
import { Table, TableBody, TableCaption, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { BASE_URL } from '@/data';
import { scanApi } from '@/api/scans';

interface ScanResult {
  id: string
  name: string
  status: 'Pass' | 'Fail'
  datetime: string
}

export default function SimplifiedScanTable() {
  const [scans, setScans] = useState<ScanResult[]>([])
  const [filters, setFilters] = useState({
    name: '',
    status: 'all'
  })

  useEffect(() => {
    const fetchScans = async () => {
      try {
        const data = await scanApi.getMultiScans() // Use scanApi here
        setScans(data)
      } catch (error) {
        console.error('Error fetching scan data:', error)
      }
    }

    fetchScans()
  }, [])

  useEffect(() => {
    // Filter scans based on user input
    const filteredScans = scans.filter(scan => {
      return (
        scan.name.toLowerCase().includes(filters.name.toLowerCase()) &&
        (filters.status === 'all' || scan.status === filters.status)
      )
    })
    setScans(filteredScans)
  }, [filters])

  const handleFilterChange = (key: 'name' | 'status', value: string) => {
    setFilters(prev => ({ ...prev, [key]: value }))
  }

  return (
    <div className="space-y-4">
      <div className="flex gap-4">
        <Input
          placeholder="Filter by Scan Name"
          value={filters.name}
          onChange={(e) => handleFilterChange('name', e.target.value)}
          className="max-w-sm"
        />
        <Select
          value={filters.status}
          onValueChange={(value) => handleFilterChange('status', value)}
        >
          <SelectTrigger className="max-w-[180px]">
            <SelectValue placeholder="Filter by Status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All</SelectItem>
            <SelectItem value="Pass">Pass</SelectItem>
            <SelectItem value="Fail">Fail</SelectItem>
          </SelectContent>
        </Select>
      </div>
      {scans.length > 0 ? (
        <Table className="min-w-full border-collapse">
          <TableHeader>
            <TableRow>
              <TableHead className="border-b">Scan Name</TableHead>
              <TableHead className="border-b">Status</TableHead>
              <TableHead className="border-b">Date & Time</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {scans.map((scan) => (
              <TableRow key={scan.id}>
                <TableCell>{scan.name}</TableCell>
                <TableCell>
                  <span className={`px-2 py-1 rounded ${scan.status === 'Pass' ? 'bg-green-500 text-white' : 'bg-red-500 text-white'}`}>
                    {scan.status}
                  </span>
                </TableCell>
                <TableCell>{new Date(scan.datetime).toLocaleString()}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      ) : (
        <div className="text-center py-8">
          <p className="text-lg font-medium text-gray-900">No results</p>
          <p className="mt-1 text-sm text-gray-500">Please try again</p>
        </div>
      )}
    </div>
  )
}
