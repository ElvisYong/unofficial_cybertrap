import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

export default function InputFile() {
  return (
    <div className="grid w-full max-w-sm items-center gap-1.5">
      <Label htmlFor="picture">Upload Domains (.txt file)</Label>
      <Input id="picture" type="file" />
    </div>
  )
}
