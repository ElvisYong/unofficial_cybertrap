"use client"

import { useState, useEffect } from 'react'
import { BarChart, Bar, PieChart, Pie, Cell, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { ExternalLink } from 'lucide-react'


interface PipelineJob {
  created_at: string
  id: number
  iid: number
  name: string
  project_id: string
  ref: string
  sha: string
  source: number
  status: string
  updated_at: string
  web_url: string
}

export default function PipelineStatusDashboard() {
  const [pipelineData, setPipelineData] = useState<PipelineJob[]>([])
  const [statusCounts, setStatusCounts] = useState<{ [key: string]: number }>({})
  const [pipelineDetails, setPipelineDetails] = useState<PipelineJob[]>([]);
  const [pipelineIndiv, setPipelineIndiv] = useState<any[]>([]);

  function setError(message: string) {
    throw new Error('Function not implemented.')
  }
  function setLoading(arg0: boolean) {
    throw new Error('Function not implemented.')
  }

  // get project pipelines 
  useEffect(() => {
    const fetchPipelineData = async () => {
        try {
          const gitlabToken = process.env.NEXT_GITLAB_TOKEN
          console.log("GitLab Token:", gitlabToken);
          const response = await fetch('https://gitlab.com/api/v4/projects/61215069/pipelines', {
            headers: {
              Authorization: `Bearer ${gitlabToken}`, // Pass the token in the Authorization header
              'Content-Type': 'application/json',
            },
          })
          if (!response.ok) {
            throw new Error(`Error: ${response.statusText}`)
          }
          const data: PipelineJob[] = await response.json()
          setPipelineData(data)
          if (pipelineData.length > 0) {
            console.log('Updated pipeline data:', pipelineData);
          }     
          
          console.log('testing', pipelineData[0])
          const counts = pipelineData.reduce((acc, job) => {
            acc[job.status] = (acc[job.status] || 0) + 1
            console.log('acc count', acc)
            return acc
          }, {} as { [key: string]: number })

        } catch (error) {
            setError((error as Error).message)
          } finally {
            setLoading(false)
          }
        console.log('pipeline data 1', pipelineData)
      }

    // if (pipelineData.length > 0) {
    //   console.log('Updated pipeline data:', pipelineData);
    // }
    console.log('testing', pipelineData[0])
    const counts = pipelineData.reduce((acc, job) => {
      acc[job.status] = (acc[job.status] || 0) + 1
      console.log('acc count', acc)
      return acc
    }, {} as { [key: string]: number })
    
    setStatusCounts(counts)
    console.log('count status', counts)

    //30days
    const currentDate = new Date(); // Get current date
    const pipelineDetails = pipelineData.reduce((acc, job) => {
      console.log('job', job)
      const jobDate = new Date(job.created_at); // Convert job.created_at to a Date object
      const timeDifference = currentDate.getTime() - jobDate.getTime();
      const differenceInDays = timeDifference / (1000 * 60 * 60 * 24);

      if (differenceInDays <=30) {
        acc.push(job);
        console.log('less than 30 days:', job.id)
      }
      return acc;
    }, [] as typeof pipelineData);
  
  

  setPipelineDetails(pipelineDetails)
  console.log('PipelineDetails', pipelineDetails)
  
  fetchPipelineData()

  }, [pipelineData])



  ///////// details of pipeline
  useEffect(() => {
    const fetchIndivPipelineData = async () => {
      try {
        const gitlabToken = process.env.NEXT_GITLAB_TOKEN;
  
        // Fetch data for each job and await the responses in parallel
        const results = await Promise.all(
          pipelineDetails.map(async (job: { id: any }) => {
            console.log('job: ', job.id);
            const response = await fetch(`https://gitlab.com/api/v4/projects/61215069/pipelines/${job.id}`, {
              headers: {
                Authorization: `Bearer ${gitlabToken}`,
                'Content-Type': 'application/json',
              },
            });
  
            if (!response.ok) {
              console.error(`Failed to fetch job with id: ${job.id}`);
              return null;
            }
  
            const data = await response.json();
            console.log('data details:', data);
            return data; // Return the fetched job data
          })
        );
  
        // Filter out null values and update the state with valid data
        setPipelineIndiv(results.filter(detail => detail !== null));
      } catch (error) {
        console.error('Error fetching pipeline details:', error);
      }
    };
  
    // Log the pipeline details after fetching
    fetchIndivPipelineData().then(() => {
      console.log('done:', pipelineIndiv);
    });
  }, [pipelineDetails]); // Add `pipelineData` as a dependency if it is updated dynamically  

  const statusColors: { [key: string]: string } = {
    'success': "success",
    'canceled': "secondary",
    'failed': "destructive",
  }
  const statusColorHex: { [key: string]: string } = {
    'success': "#28a745", // Green for success
    'canceled': "#d3d3d3", // Yellow/Orange for canceled
    'failed': "#dc3545", // Red for failed
  };
  const statusData = Object.entries(statusCounts).map(([status, count]) => ({
    status,
    count
  }))

  const RADIAN = Math.PI / 180
  const renderCustomizedLabel = ({ cx, cy, midAngle, innerRadius, outerRadius, percent, index }: any) => {
    const radius = innerRadius + (outerRadius - innerRadius) * 0.5
    const x = cx + radius * Math.cos(-midAngle * RADIAN)
    const y = cy + radius * Math.sin(-midAngle * RADIAN)

    return (
      <text x={x} y={y} fill="white" textAnchor={x > cx ? 'start' : 'end'} dominantBaseline="central">
        {`${(percent * 100).toFixed(0)}%`}
      </text>
    )
  }

  return (
    <div className="container mx-auto py-10">
      {/* <h1 className="text-3xl font-bold mb-6">Pipeline Status Dashboard</h1> */}

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mb-6">
        <div>
        <Card>
          <CardHeader>
            <CardTitle>Status Summary</CardTitle>
            <CardDescription>Distribution of pipeline job statuses</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-[300px] w-full">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie data={statusData} dataKey="count" nameKey="status" cx="50%" cy="50%" outerRadius={100} label={renderCustomizedLabel} labelLine={false}>
                    {statusData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={statusColorHex[entry.status] || "#8884d8"} />
                    ))}
                  </Pie>
                  <Tooltip />
                  <Legend />
                </PieChart>
              </ResponsiveContainer>
            </div>
          </CardContent>
        </Card>          
        </div>


        <Card>
          <CardHeader>
            <CardTitle>Status Counts</CardTitle>
            <CardDescription>Detailed count of each status</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 gap-5">
              {Object.entries(statusCounts).map(([status, count]) => (
                <div key={status} className="flex items-center justify-between">
                  <Badge variant={statusColors[status]}>
                  {/* <Badge variant="success"> */}
                    {status}
                  </Badge>
                  <span className="font-bold">{count}</span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Time Summary</CardTitle>
            <CardDescription>Overview of pipeline job duration</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-[300px]">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={statusData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="status" />
                  <YAxis allowDecimals={false} />
                  <Tooltip />
                  <Bar dataKey="count">
                    {statusData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={statusColorHex[entry.status] || "#8884d8"} />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Pipelines</CardTitle>
              <CardDescription>Last 30 days</CardDescription>          
            </div>
            <a
              href="https://gitlab.com/cybertrap4831/cybertrap/-/pipelines"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center text-blue-500 hover:text-blue-700 transition-colors"
              aria-label="Open GitLab Pipelines in new tab"
            >
              <ExternalLink className="h-5 w-5 mr-2" />
              View All Pipelines in GitLab
            </a>

          </div>        
          </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>ID</TableHead>
                <TableHead>Name</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Stage</TableHead>
                <TableHead>Duration</TableHead>
                <TableHead>Started At</TableHead>
                <TableHead>Updated At</TableHead>
                <TableHead>View in GitLab</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {pipelineIndiv.map((job) => (
                <TableRow key={job.id}>
                  <TableCell>{job.id}</TableCell>
                  <TableCell>{job.name}</TableCell>
                  <TableCell>
                    <Badge variant={statusColors[job.status]}>
                      {job.status}
                    </Badge>
                  </TableCell>
                  <TableCell>{job.status}</TableCell>
                   <TableCell>{job.duration}s</TableCell>
                  <TableCell>{new Date(job.started_at).toLocaleString()}</TableCell>
                  <TableCell>{job.updated_at ? new Date(job.finished_at).toLocaleString() : 'N/A'}</TableCell>
                  <TableCell>
                    <a
                      href={job.web_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex items-center text-blue-500 hover:text-blue-700 transition-colors"
                      aria-label="Open GitLab Pipelines in new tab"
                    >
                      <ExternalLink className="h-5 w-5 mr-2" />
                    </a>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}

