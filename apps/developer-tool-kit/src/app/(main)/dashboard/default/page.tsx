import { NetworkOverview } from "./_components/network-overview";

export default function Page() {
  return (
    <div className="@container/main flex flex-col gap-4 md:gap-6">
      <NetworkOverview />
    </div>
  );
}
