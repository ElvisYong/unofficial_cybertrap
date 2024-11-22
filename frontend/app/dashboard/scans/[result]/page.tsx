"use client";

// import { useRouter } from 'next/navigation';
import { useParams } from 'next/navigation';

import React, { useEffect, useState } from 'react';
import DashboardCard from "@/components/dashboard/DashboardCard";;
import AnalyticsChart from '@/components/dashboard/AnalyticsChart';
import { scanApi } from '@/api/scans';
import { Scan } from '@/app/types';
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { ExternalLink } from 'lucide-react';

const scans =
{ 'domain': "sandboxed.com",
  'domainId': "673506f0eaa3e2eef6abe49b",
  'error': "<nil>",
  'id': "67360e51eaa3e2eef6abe4e7",
  'scanDate': "2024-11-14",
  'scanTook': 81669,
  's3ResultURL': [
    "https://cybertrap-scan-results.s3.ap-southeast-1.amazonaws.com/nameserver-fingerprint__1731600105047.json",
    "https://cybertrap-scan-results.s3.ap-southeast-1.amazonaws.com/caa-fingerprint__1731600105156.json"],
  'status': "complete"}

  const statusColors: { [key: string]: string } = {
    'complete': "success",
    'canceled': "secondary",
    'failed': "destructive",
  }

const TargetDetailPage: React.FC = () => {
//   const router = useRouter();
//   const { target } = router.target;  // Extract the target name from the URL
    const params = useParams();
    const target: string | string[] = params.result;
    const targetArray: string[] = Array.isArray(target) ? target : [target];
    console.log('params', params)
    console.log('target', target)

    interface PipelineJobDetails {
      domain: string
      domainId: number
      error: string
      id: number
      scanDate: string
      scanTook: number
      status: string
    }
    
    // const [scans, setScans] = useState<any[]>([])
    const [scanDetails, setScanDetails] = useState<any[]>([]);

    useEffect(() => {
      const fetchScans = async () => {
        try {
          const data = await scanApi.getAllScans(); // Use scanApi to fetch scans
          console.log('data', data)
          const details = await scanApi.getScanById(targetArray);
          setScanDetails(details)
          console.log('detail', details)
  
          const pipelineDetails = details.reduce((acc, job) => {
            console.log('job', job)
            return acc;
          }, [] as unknown as typeof details)
  
          setScanDetails(details)
          const sortedScans = data.sort((a: Scan, b: Scan) => 
            new Date(b.scanDate).getTime() - new Date(a.scanDate).getTime()
          )
          console.log('detail1', details)
  
      
          setScans(sortedScans)
          // setFilteredScans(sortedScans)
          
        } catch (error) {
          console.error('Error fetching scans:', error)
        }
      }  

      fetchScans()
    }, [])
    

  
  return (
    <>
    <div className="container mx-auto px-4 py-8">
      <Card className="mb-8">
        <CardHeader>
        <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center">
            <CardTitle className="text-2xl font-bold">Scan Summary</CardTitle>
        </div>
        <div>
          <Badge variant={statusColors[scans.status]}>{scans.status}</Badge>
        </div>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            <p className="text-lg"><strong>Target Name:</strong> {scans.domain}</p>
            <p className="text-lg"><strong>Target ID:</strong> {target}</p>
          </div>
        </CardContent>
      
        <div className='grid grid-cols-1 sm:grid-cols-2 gap-6 mb-8'>
        <DashboardCard
          title='Scan Duration'
          count={scans.scanTook}
        />
        <DashboardCard
          title='Results Count'
          count={scans.s3ResultURL.length}
        />   
      </div>

        </Card>
      <Card>
        <CardHeader>
          <CardTitle>Scan Results</CardTitle>
        </CardHeader>
        <CardContent>
          <ul className="space-y-2">
            {scans.s3ResultURL.map((url, index) => (
              <li key={index} className="flex items-center">
                  <a
                    href={url} 
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex items-center text-blue-500 hover:text-blue-700 transition-colors"
                    aria-label="Open GitLab Pipelines in new tab"
                  >
                    <ExternalLink className="h-5 w-5 mr-2" />
                    {url}
                  </a>
              </li>
            ))}
          </ul>
        </CardContent>
      </Card>
      {/* <div> 
        <AnalyticsChart />
      </div>    */}
    </div>
    </>
  );
};

export default TargetDetailPage;


// function setFilteredScans(sortedScans: Scan[]) {
//   throw new Error('Function not implemented.');
// }

// function setDetails(scanDetails: void) {
//   throw new Error('Function not implemented.');
// }
// export default function TargetDetailPage({ params }: { params: {result: string} }) {
//   return (
//     <div className="max-w-3xl mx-auto mt-8">
//       <h1 className="text-2xl font-bold">Scan Summary</h1>
//       <p className="mt-4">Target Name: { result } </p>
//       {/* Add more details about the target as needed */}
//     </div>
//   );
// }