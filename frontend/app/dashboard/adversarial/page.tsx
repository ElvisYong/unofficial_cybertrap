import AdversarialTable from '../../ui/adversarial/table';

export default function AdversarialScans() {
    return (
      <div className="container mx-auto p-4">
        <h1 className="text-2xl font-bold mb-4">Adversarial Scan History</h1>
        <AdversarialTable/>
      </div>
    )
  }