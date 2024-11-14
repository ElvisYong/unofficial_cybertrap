import OverallScansTable from '../../ui/overall/table';

export default function OverallScans() {
    return (
      <div className="container mx-auto p-4">
        <h1 className="text-2xl font-bold mb-4">Overall Scan History</h1>
        <OverallScansTable/>
      </div>
    )
  }