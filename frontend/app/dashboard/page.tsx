import ScheduleScan from '../ui/schedule/scheduleScanForm';
import ScheduleTable from '../ui/schedule/table';

export default function ScheduleScanPage() {
  return (
    <div className="container mx-auto p-4">
      <div className='grid grid-cols-1 lg:grid-cols-2 gap-6 vh-100'>
        <div className='col-lg-6'>
          <h1 className="text-2xl font-bold mb-4">Schedule Scan</h1>
          <ScheduleScan />
        </div>
        <div className='py-10 lg:py-0'>
          <h1 className="text-2xl font-bold mb-4">Scheduled Scans</h1>
          <ScheduleTable />
        </div>
      </div>
    </div>
  );
}