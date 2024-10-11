import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
  } from "@/components/ui/select"

// Filter By Status Component
type FilterProps = {
    filterType : string;
    placeholder : string;
    onFilter: (filterType: string, filterValue: string) => void;
    value: string;
};

const FilterByDropdown: React.FC<FilterProps> = ({ onFilter, value }) => {
  return (
    <div className="mb-4">
      <Select value={value} onValueChange={(value) => onFilter("status", value)}>
        <SelectTrigger>
          <SelectValue placeholder="Filter by Status" />
        </SelectTrigger>
        <SelectContent>
        <SelectItem value="completed">Completed</SelectItem>
        <SelectItem value="in-progress">In Progress</SelectItem>
        <SelectItem value="pending">Pending</SelectItem>
        <SelectItem value="failed">Failed</SelectItem>
        </SelectContent>
      </Select>
    </div>
  );
};

export default FilterByDropdown