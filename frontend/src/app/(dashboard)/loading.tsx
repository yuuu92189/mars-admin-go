import { Skeleton } from "@/components/ui/surface";

export default function DashboardLoading() {
  return (
    <div className="page">
      <div className="page-head">
        <Skeleton width="220px" height={28} />
      </div>
      <div className="bento-grid">
        <div className="bento bento-hero">
          <Skeleton width="100%" height={220} />
        </div>
        <div className="bento bento-side">
          <Skeleton width="70%" height={20} />
        </div>
      </div>
    </div>
  );
}
