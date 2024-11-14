'use client'
import React, { useState, useEffect, useMemo } from 'react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from "@/components/ui/table"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Accordion, AccordionItem, AccordionTrigger, AccordionContent } from "@/components/ui/accordion"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { scanApi } from '@/api/scans';

interface ScanResult {
  id: string
  name: string
  status: 'Pass' | 'Fail' | 'in-progress'
  datetime: string
  totalScans: string
  completedScans: string
  failedScans: string
  scanIds: string[] 
}

export default function SimplifiedScanTable() {
  const [scans, setScans] = useState<ScanResult[]>([])
  const [filteredScans, setFilteredScans] = useState<ScanResult[]>([])
  const [filters, setFilters] = useState({
    name: '',
    status: 'all'
  })

  useEffect(() => {
    const fetchScans = async () => {
      try {
        const data = await scanApi.getMultiScans()
        setScans(data)
        setFilteredScans(data) // Initialize with the full data set
      } catch (error) {
        console.error('Error fetching scan data:', error)
      }
    }

    fetchScans()
  }, [])

  useEffect(() => {
    // Reset to original data if no filter is applied
    if (filters.name === '' && filters.status === 'all') {
      setFilteredScans(scans)
    } else {
      // Filter scans based on user input
      const result = scans.filter(scan => {
        return (
          scan.name.toLowerCase().includes(filters.name.toLowerCase()) &&
          (filters.status === 'all' || scan.status === filters.status)
        )
      })
      setFilteredScans(result)
    }
  }, [filters, scans])

  const handleFilterChange = (key: 'name' | 'status', value: string) => {
    setFilters(prev => ({ ...prev, [key]: value }))
  }

  const summary = useMemo(() => {
    const totalScans = filteredScans.length
    const passedScans = filteredScans.filter(scan => scan.status === 'Pass').length
    const failedScans = filteredScans.filter(scan => scan.status === 'Fail').length
  
    // Calculate pass and fail percentages
    const passPercentage = totalScans > 0 ? (passedScans / totalScans) * 100 : 0
    const failPercentage = totalScans > 0 ? (failedScans / totalScans) * 100 : 0
  
    return {
      totalScans,
      passedScans,  
      failedScans,  
      passPercentage: passPercentage.toFixed(2),  
      failPercentage: failPercentage.toFixed(2)   
    }
  }, [filteredScans])


  return (
    <div className="space-y-4">
      <div className="grid gap-4 md:grid-cols-3">
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium">Total Scans</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">{summary.totalScans}</div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium">Overall Pass Percentage</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold text-green-600">{summary.passPercentage}%</div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium">Overall Fail Percentage</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold text-red-600">{summary.failPercentage}%</div>
        </CardContent>
      </Card>
    </div>

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
            <SelectItem value="in-progress">In Progress</SelectItem>
          </SelectContent>
        </Select>
      </div>
      {filteredScans.length > 0 ? (
        <Table className="min-w-full border-collapse">
          <TableHeader>
            <TableRow>
              <TableHead className="border-b">Scan Name</TableHead>
              <TableHead className="border-b">Status</TableHead>
              <TableHead className="border-b">Date & Time</TableHead>
              <TableHead className="border-b">Total Scans</TableHead>
              <TableHead className="border-b">Completed Scans</TableHead>
              <TableHead className="border-b">Failed Scans</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {filteredScans.map((scan) => (
              <React.Fragment key={scan.id}>
                {/* Main Row */}
                <TableRow>
                  <TableCell>{scan.name}</TableCell>
                  <TableCell>
                    <span className={`px-2 py-1 rounded ${scan.status === 'Pass' ? 'bg-green-500 text-white' : scan.status === 'Fail' ? 'bg-red-500 text-white' : 'bg-yellow-500 text-black'}`}>
                      {scan.status}
                    </span>
                  </TableCell>
                  <TableCell>{new Date(scan.datetime).toLocaleString()}</TableCell>
                  <TableCell>
                  <span className="font-bold inline-block px-3 py-1 rounded-full bg-gray-200 text-gray-700">
                    {scan.totalScans}
                  </span>
                </TableCell>
                <TableCell>
                  <span className="font-bold inline-block px-3 py-1 rounded-full bg-green-200 text-green-700">
                    {scan.completedScans}
                  </span>
                </TableCell>
                <TableCell>
                  <span className="font-bold inline-block px-3 py-1 rounded-full bg-red-200 text-red-700">
                    {scan.failedScans}
                  </span>
                </TableCell>
                  <TableCell>
                    <Accordion type="single" collapsible>
                      <AccordionItem value={`details-${scan.id}`}>
                        <AccordionTrigger className="text-green-600 cursor-pointer">
                          View Details
                        </AccordionTrigger>
                        <AccordionContent>
                          {/* Accordion Content Row */} 
                          <TableRow>
                          <TableCell colSpan={7} className="bg-gray-100" style={{ minWidth: '350px' }}>
                            <div className="flex flex-col gap-2 p-4">
                              <p className="text-gray-700 font-medium">Scan IDs:</p>
                              <div className="overflow-y-auto max-h-40">
                                <ul className="list-decimal list-inside pl-5">
                                  {/* Each scan ID will be on its own line, and no overflow */}
                                  {scan.scanIds.map((id) => (
                                    <li key={id} className="break-words">
                                      <a href={`http://localhost:3000/dashboard/scans/${id}`} className="text-green-600 hover:underline">
                                        {id}
                                      </a>
                                    </li>
                                  ))}
                                </ul>
                              </div>
                            </div>
                          </TableCell>
                        </TableRow>
                        </AccordionContent>
                      </AccordionItem>
                    </Accordion>
                  </TableCell>
                </TableRow>
              </React.Fragment>
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
