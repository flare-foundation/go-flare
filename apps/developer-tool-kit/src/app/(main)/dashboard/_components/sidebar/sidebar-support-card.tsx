import Link from "next/link";

import { ExternalLink } from "lucide-react";

import { Card, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { APP_CONFIG } from "@/config/app-config";

export function SidebarSupportCard() {
  return (
    <Card size="sm" className="shadow-none group-data-[collapsible=icon]:hidden">
      <CardHeader className="px-4">
        <CardTitle className="text-sm">Titan Local UAT</CardTitle>
        <CardDescription>
          Chain ID {APP_CONFIG.titan.chainIdDec} · Native token {APP_CONFIG.titan.nativeToken.symbol}.{" "}
          <Link
            href={APP_CONFIG.titan.explorerUrl}
            className="inline-flex items-center gap-0.5 text-foreground underline-offset-4 hover:underline"
          >
            Open explorer
            <ExternalLink className="size-3" aria-hidden />
          </Link>
        </CardDescription>
      </CardHeader>
    </Card>
  );
}