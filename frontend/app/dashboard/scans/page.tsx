import ScanResultsTable from '../../ui/scans/table';

export default function ScanResultsPage() {
  return (
    <div className="container mx-auto p-4">
      <h1 className="text-2xl font-bold mb-4">Scan Results (by Domain)</h1>
      <ScanResultsTable />
    </div>
  )
}