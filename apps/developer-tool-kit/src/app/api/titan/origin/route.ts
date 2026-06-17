import { type NextRequest, NextResponse } from "next/server";

import { readOriginFile } from "@/lib/titan/origin";

export const dynamic = "force-dynamic";

export async function GET(request: NextRequest) {
  const { searchParams } = request.nextUrl;
  const format = searchParams.get("format");

  try {
    const summary = readOriginFile();

    if (format === "raw") {
      return new NextResponse(JSON.stringify(summary.origin, null, 2), {
        headers: {
          "Content-Type": "application/json",
          "Content-Disposition": 'attachment; filename="origin.json"',
          "Cache-Control": "no-store",
        },
      });
    }

    return NextResponse.json({
      ...summary,
      apiRawUrl: "/api/titan/origin?format=raw",
      dockerEnv: {
        ORIGIN_URL: summary.githubRawUrl,
        GENESIS_FILE: "/app/titan/origin.json",
        TITAN_NETWORK_ID: String(summary.origin.networkID),
      },
    });
  } catch (error) {
    return NextResponse.json(
      { error: "Failed to load origin.json", details: String(error) },
      { status: 500 },
    );
  }
}